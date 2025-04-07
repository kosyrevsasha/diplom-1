package repository

import (
	"bufio"
	"database/sql"
	"diplom-1/cmd/gophermart/internal/config"
	"errors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"log"
	"math"
	"net/http"
	"os"
	"time"
)

var (
	ErrConflict       = errors.New("логин занят")
	ErrBadCredentials = errors.New("неверная пара логин/пароль")
)

type User struct {
	ID int `json:"id"`
	Credentials
}

type Order struct {
	ID         string    `json:"number"`
	Status     string    `json:"status"`
	Accrual    float64   `json:"accrual,omitempty"`
	UploadedAt time.Time `json:"uploaded_at"`
}

type UserOrder struct {
	UserID  int    `json:"user_id"`
	OrderID string `json:"order_id"`
}

type Balance struct {
	Current   float64 `json:"current"`
	Withdrawn float64 `json:"withdrawn"`
}

type Withdrawal struct {
	Order       string    `json:"order"`
	Sum         float64   `json:"sum"`
	ProcessedAt time.Time `json:"processed_at"`
}

const (
	NEW        = "NEW"
	PROCESSING = "PROCESSING"
	INVALID    = "INVALID"
	PROCESSED  = "PROCESSED"
)

type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

type Credentials struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type WithdrawRequest struct {
	Order string  `json:"order"`
	Sum   float64 `json:"sum"`
}

type ProcessedOrder struct {
	Number  string  `json:"order"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
}

type Repository interface {
	CreateUser(Credentials) (User, error)
	FindUser(string) (User, error)
	FindUserOrders(int) ([]Order, error)
	SaveOrder(int, string) (int, error)
	UpdateOrder(ProcessedOrder) error
	InvalidateOrder(string) error
	GetUserBalance(int) (Balance, error)
	MakeWithdraw(string, float64, int) (int, error)
	GetUserWithdrawals(int) ([]Withdrawal, error)
}

func InitDB() (Repository, error) {
	db := DB{}
	db.Init()
	return &db, nil
}

type DB struct {
	Repository
}

func (db *DB) Init() {
	pgdb := db.getPgdb()
	defer pgdb.Close()

	sqlFile, err := os.OpenFile("./cmd/gophermart/internal/repository/db.sql", os.O_RDONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	var script string
	scanner := bufio.NewScanner(sqlFile)
	for scanner.Scan() {
		bytes := scanner.Bytes()
		script += string(bytes)
	}
	sqlFile.Close()

	if _, sqlErr := pgdb.Exec(script); sqlErr != nil {
		log.Fatal(sqlErr)
	}
}

func (db *DB) getPgdb() *sql.DB {
	pgdb, err := sql.Open("pgx", config.ProcessConfig.DataBase)
	if err != nil {
		log.Fatal(err)
	}
	return pgdb
}

func (db *DB) CreateUser(creds Credentials) (User, error) {
	pgdb := db.getPgdb()
	defer pgdb.Close()
	var pgErr *pgconn.PgError

	_, qErr := pgdb.Exec("INSERT INTO users (login, password) VALUES ($1, $2);", creds.Login, creds.Password)
	if qErr != nil {
		if errors.As(qErr, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return User{}, ErrConflict
		} else {
			return User{}, qErr
		}
	}

	row := pgdb.QueryRow("SELECT * FROM users WHERE login = $1;", creds.Login)

	var user User
	errScan := row.Scan(&user.ID, &user.Login, &user.Password)
	if errScan != nil || row == nil {
		log.Println(errScan)
		return User{}, errors.New("user not found")
	}
	return user, nil
}

func (db *DB) FindUser(login string) (User, error) {
	pgdb := db.getPgdb()
	defer pgdb.Close()
	var user User

	row := pgdb.QueryRow("SELECT * FROM users WHERE login = $1;", login)
	errScan := row.Scan(&user.ID, &user.Login, &user.Password)
	if errScan != nil || row == nil {
		log.Println(errScan)
		return User{}, errors.New("неверная пара логин/пароль")
	}

	return user, nil
}

func (db *DB) FindUserOrders(id int) ([]Order, error) {
	pgdb := db.getPgdb()
	defer pgdb.Close()

	query := "SELECT o.id as number, o.status, o.accrual, o.uploaded_at  FROM orders o " +
		"LEFT JOIN user_orders uo ON uo.order_id = o.id " +
		"WHERE uo.user_id = $1 " +
		"ORDER BY o.uploaded_at DESC"

	rows, qerr := pgdb.Query(query, id)
	if qerr != nil {
		return nil, qerr
	}
	err := rows.Err()
	if err != nil {
		return nil, qerr
	}
	defer rows.Close()

	var orders []Order

	for rows.Next() {
		var order Order
		var accrual sql.NullFloat64
		err := rows.Scan(&order.ID, &order.Status, &accrual, &order.UploadedAt)
		if err != nil {
			return nil, err
		}
		if accrual.Valid {
			order.Accrual = roundFloat(accrual.Float64, 2)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (db *DB) SaveOrder(userID int, orderNum string) (int, error) {
	pgdb := db.getPgdb()
	defer pgdb.Close()
	var pgErr *pgconn.PgError

	row := pgdb.QueryRow("SELECT order_id, user_id FROM user_orders WHERE order_id = $1", orderNum)

	userOrder := UserOrder{}
	sErr := row.Scan(&userOrder.OrderID, &userOrder.UserID)
	if !errors.Is(sErr, sql.ErrNoRows) {
		if userOrder.UserID == userID {
			return http.StatusOK, errors.New("номер заказа уже был загружен этим пользователем")
		} else {
			return http.StatusConflict, errors.New("номер заказа уже был загружен другим пользователем")
		}
	}

	_, qErr := pgdb.Exec("INSERT INTO orders (id, status) VALUES ($1, $2)", orderNum, PROCESSING)
	if qErr != nil {
		if errors.As(qErr, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return http.StatusConflict, qErr
		} else {
			return http.StatusInternalServerError, qErr
		}
	}

	_, qErr = pgdb.Exec("INSERT INTO user_orders (user_id, order_id) VALUES ($1, $2)", userID, orderNum)
	if qErr != nil {
		return http.StatusInternalServerError, qErr
	}

	return http.StatusAccepted, nil
}

func (db *DB) UpdateOrder(order ProcessedOrder) error {
	pgdb := db.getPgdb()
	defer pgdb.Close()
	_, qErr := pgdb.Exec("UPDATE orders SET accrual = $1, status = $2 WHERE id = $3", order.Accrual, order.Status, order.Number)
	if qErr != nil {
		return qErr
	}

	return nil
}

func (db *DB) InvalidateOrder(orderNum string) error {
	pgdb := db.getPgdb()
	defer pgdb.Close()
	_, qErr := pgdb.Exec("UPDATE orders SET status = $1 WHERE id = $2", INVALID, orderNum)
	if qErr != nil {
		return qErr
	}

	return nil
}

func (db *DB) GetUserBalance(userID int) (Balance, error) {
	pgdb := db.getPgdb()
	defer pgdb.Close()

	baseQuery := "SELECT SUM(o.accrual) FROM orders o " +
		"LEFT JOIN user_orders uo ON uo.order_id = o.id " +
		"WHERE o.status = 'PROCESSED' " +
		"AND uo.user_id = $1 " +
		"AND o.withdrawal IS "

	query := "SELECT (" + baseQuery + "FALSE " + ") as accruals, (" + baseQuery + "TRUE" + ") as withdrawals"

	row := pgdb.QueryRow(query, userID)
	var (
		accruals    sql.NullFloat64
		withdrawals sql.NullFloat64
	)

	err := row.Scan(&accruals, &withdrawals)
	if err != nil {
		return Balance{}, err
	}
	currentBalance := accruals.Float64 - withdrawals.Float64
	b := Balance{roundFloat(currentBalance, 2), withdrawals.Float64}
	return b, nil
}

func (db *DB) MakeWithdraw(orderNum string, sum float64, userID int) (int, error) {
	pgdb := db.getPgdb()
	defer pgdb.Close()
	var pgErr *pgconn.PgError

	_, qErr := pgdb.Exec("INSERT INTO orders (id, accrual, status, withdrawal) VALUES ($1, $2, $3, $4)", orderNum, sum, PROCESSED, true)
	if qErr != nil {
		if errors.As(qErr, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return http.StatusConflict, errors.New("номер заказа уже существует")
		} else {
			return http.StatusInternalServerError, qErr
		}
	}

	_, qErr = pgdb.Exec("INSERT INTO user_orders (user_id, order_id) VALUES ($1, $2)", userID, orderNum)
	if qErr != nil {
		return http.StatusInternalServerError, qErr
	}

	return http.StatusOK, nil
}

func (db *DB) GetUserWithdrawals(userID int) ([]Withdrawal, error) {
	pgdb := db.getPgdb()
	defer pgdb.Close()

	query := "SELECT o.id, o.accrual, o.uploaded_at  FROM orders o " +
		"LEFT JOIN user_orders uo ON uo.order_id = o.id " +
		"WHERE uo.user_id = $1 AND o.withdrawal IS TRUE " +
		"ORDER BY o.uploaded_at DESC"

	rows, qerr := pgdb.Query(query, userID)
	if qerr != nil {
		return nil, qerr
	}
	err := rows.Err()
	if err != nil {
		return nil, qerr
	}
	defer rows.Close()
	wdrs := []Withdrawal{}

	for rows.Next() {
		var w Withdrawal
		var accrual sql.NullFloat64
		err := rows.Scan(&w.Order, &accrual, &w.ProcessedAt)
		if err != nil {
			return nil, err
		}
		if accrual.Valid {
			w.Sum = roundFloat(accrual.Float64, 2)
		}
		wdrs = append(wdrs, w)
	}

	return wdrs, nil
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
