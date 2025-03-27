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
	Id int `json:"id"`
	Credentials
}

type Order struct {
	Id      int     `json:"number"`
	Status  string  `json:"status"`
	Accrual float64 `json:"accrual,omitempty"`
	// TODO: преобразоывыать отдельно?
	UploadedAt time.Time `json:"uploaded_at"`
}

type UserOrder struct {
	UserId  int `json:"user_id"`
	OrderId int `json:"order_id"`
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

type Repository interface {
	CreateUser(Credentials) (User, error)
	FindUser(Credentials) (User, error)
	FindUserOrders(int) ([]Order, error)
	SaveOrder(int, int) (int, error)
	GetUserBalance(int) (Balance, error)
	MakeWithdraw(int, float64, int) (int, error)
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

	sqlFile, err := os.OpenFile("./internal/repository/db.sql", os.O_RDONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	var script string
	scanner := bufio.NewScanner(sqlFile)
	for ok := scanner.Scan(); ok != false; ok = scanner.Scan() {
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
	errScan := row.Scan(&user.Id, &user.Login, &user.Password)
	if errScan != nil || row == nil {
		log.Println(errScan)
		return User{}, errors.New("user not found")
	}
	return user, nil
}

func (db *DB) FindUser(creds Credentials) (User, error) {
	pgdb := db.getPgdb()
	defer pgdb.Close()
	var user User

	row := pgdb.QueryRow("SELECT * FROM users WHERE login = $1 AND password = $2;", creds.Login, creds.Password)
	errScan := row.Scan(&user.Id, &user.Login, &user.Password)
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
		"WHERE uo.user_id = $1 AND o.withdrawal IS FALSE " +
		"ORDER BY o.uploaded_at DESC"

	rows, qerr := pgdb.Query(query, id)
	defer rows.Close()
	if qerr != nil {
		return nil, qerr
	}
	orders := []Order{}

	for rows.Next() {
		var order Order
		var accrual sql.NullFloat64
		err := rows.Scan(&order.Id, &order.Status, &accrual, &order.UploadedAt)
		if err != nil {
			return nil, err
		}
		if accrual.Valid {
			order.Accrual = roundFloat(accrual.Float64, 1)
		}
		orders = append(orders, order)
	}

	return orders, nil
}

func (db *DB) SaveOrder(orderNum int, userId int) (int, error) {
	pgdb := db.getPgdb()
	defer pgdb.Close()
	var pgErr *pgconn.PgError

	row := pgdb.QueryRow("SELECT order_id, user_id FROM user_orders WHERE order_id = $1", orderNum)

	userOrder := UserOrder{}
	sErr := row.Scan(&userOrder.OrderId, &userOrder.UserId)
	if !errors.Is(sql.ErrNoRows, sErr) {
		if userOrder.UserId == userId {
			return http.StatusOK, errors.New("номер заказа уже был загружен этим пользователем")
		} else {
			return http.StatusConflict, errors.New("номер заказа уже был загружен другим пользователем")
		}
	}

	_, qErr := pgdb.Exec("INSERT INTO orders (id, status) VALUES ($1, $2)", orderNum, NEW)
	if qErr != nil {
		if errors.As(qErr, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return http.StatusConflict, nil
		} else {
			return http.StatusInternalServerError, qErr
		}
	}

	_, qErr = pgdb.Exec("INSERT INTO user_orders (user_id, order_id) VALUES ($1, $2)", userId, orderNum)
	if qErr != nil {
		return http.StatusInternalServerError, qErr
	}

	return http.StatusAccepted, nil
}

func (db *DB) GetUserBalance(userId int) (Balance, error) {
	pgdb := db.getPgdb()
	defer pgdb.Close()

	baseQuery := "SELECT SUM(o.accrual) FROM orders o " +
		"LEFT JOIN user_orders uo ON uo.order_id = o.id " +
		"WHERE o.status = 'PROCESSED' " +
		"AND uo.user_id = $1 " +
		"AND o.withdrawal IS "

	query := "SELECT (" + baseQuery + "FALSE " + ") as accruals, (" + baseQuery + "TRUE" + ") as withdrawals"

	row := pgdb.QueryRow(query, userId)
	var (
		accruals    sql.NullFloat64
		withdrawals sql.NullFloat64
	)

	err := row.Scan(&accruals, &withdrawals)
	if err != nil {
		return Balance{}, err
	}
	currentBalance := accruals.Float64 - withdrawals.Float64
	b := Balance{roundFloat(currentBalance, 1), withdrawals.Float64}
	return b, nil
}

func (db *DB) MakeWithdraw(orderNum int, sum float64, userId int) (int, error) {
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

	_, qErr = pgdb.Exec("INSERT INTO user_orders (user_id, order_id) VALUES ($1, $2)", userId, orderNum)
	if qErr != nil {
		return http.StatusInternalServerError, qErr
	}

	return http.StatusOK, nil
}

func (db *DB) GetUserWithdrawals(userId int) ([]Withdrawal, error) {
	pgdb := db.getPgdb()
	defer pgdb.Close()

	query := "SELECT o.id, o.accrual, o.uploaded_at  FROM orders o " +
		"LEFT JOIN user_orders uo ON uo.order_id = o.id " +
		"WHERE uo.user_id = $1 AND o.withdrawal IS TRUE " +
		"ORDER BY o.uploaded_at DESC"

	rows, qerr := pgdb.Query(query, userId)
	defer rows.Close()
	if qerr != nil {
		return nil, qerr
	}
	wdrs := []Withdrawal{}

	for rows.Next() {
		var w Withdrawal
		var accrual sql.NullFloat64
		err := rows.Scan(&w.Order, &accrual, &w.ProcessedAt)
		if err != nil {
			return nil, err
		}
		if accrual.Valid {
			w.Sum = roundFloat(accrual.Float64, 1)
		}
		wdrs = append(wdrs, w)
	}

	return wdrs, nil
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
