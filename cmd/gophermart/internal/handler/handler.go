package handler

import (
	"diplom-1/cmd/gophermart/internal/accrual"
	"diplom-1/cmd/gophermart/internal/repository"
	"encoding/json"
	"errors"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/theplant/luhn"
	"io"
	"log"
	"net/http"
	"strconv"
)

func Register(db repository.Repository, w http.ResponseWriter, r *http.Request) {
	var creds repository.Credentials
	err := ReadRequestData(r, &creds)
	if err != nil {
		MakeErrResponse(&w, http.StatusBadRequest, "неверный формат запроса")
		return
	}

	if creds.Login == "" || creds.Password == "" {
		MakeErrResponse(&w, http.StatusBadRequest, "логин или пароль не указаны")
		return
	}

	creds.Password = HashPassword(creds.Password)

	user, err := db.CreateUser(creds)
	if err != nil {
		if errors.Is(err, repository.ErrConflict) {
			w.WriteHeader(http.StatusConflict)
		} else {
			MakeErrResponse(&w, http.StatusInternalServerError, "")
		}
		w.Write([]byte(err.Error()))

		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(strconv.Itoa(user.Id)))
}

func Login(db repository.Repository, w http.ResponseWriter, r *http.Request) {
	var creds repository.Credentials
	err := ReadRequestData(r, &creds)
	if err != nil {
		MakeErrResponse(&w, http.StatusBadRequest, "неверный формат запроса")
		return
	}

	if creds.Login == "" || creds.Password == "" {
		MakeErrResponse(&w, http.StatusBadRequest, "логин или пароль не указаны")
		return
	}
	creds.Password = HashPassword(creds.Password)
	user, err := db.FindUser(creds)

	if err != nil {
		var code int
		if errors.Is(err, repository.ErrBadCredentials) {
			code = http.StatusUnauthorized
		} else {
			code = http.StatusInternalServerError
		}
		MakeErrResponse(&w, code, err.Error())
		return
	}
	// ---- JWT ----
	newToken, tErr := BuildJWTString(user.Id)
	if tErr != nil {
		MakeErrResponse(&w, http.StatusInternalServerError, "")
		log.Println(tErr)
		return
	}
	w.Header().Set("Authorization", "Bearer "+newToken)
	w.WriteHeader(http.StatusOK)
	// ---- ----
}

func ProcessOrder(db repository.Repository, w http.ResponseWriter, r *http.Request, worker *accrual.Worker) {
	userId := getUserId(r)

	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil || len(body) == 0 {
		MakeErrResponse(&w, http.StatusBadRequest, err.Error())
		return
	}

	number, convErr := strconv.Atoi(string(body))
	if convErr != nil {
		MakeErrResponse(&w, http.StatusBadRequest, convErr.Error())
		return
	}

	// TODO: проверять
	if !luhn.Valid(number) {
		MakeErrResponse(&w, http.StatusUnprocessableEntity, "неверный формат номера заказа")
		return
	}
	orderNum := strconv.Itoa(number)
	code, saveErr := db.SaveOrder(orderNum, userId)
	if saveErr != nil {
		MakeErrResponse(&w, code, saveErr.Error())
		return
	}

	order := buildOrder(orderNum)
	accrual.RegisterOrder(order)
	worker.CheckChanel <- orderNum

	w.WriteHeader(code)
}

func UserOrders(db repository.Repository, w http.ResponseWriter, r *http.Request) {
	orders, err := db.FindUserOrders(getUserId(r))
	if err != nil {
		MakeErrResponse(&w, http.StatusInternalServerError, err.Error())
		return
	}

	res, marshErr := json.Marshal(orders)
	if marshErr == nil {
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func UserBalance(db repository.Repository, w http.ResponseWriter, r *http.Request) {
	balance, err := db.GetUserBalance(getUserId(r))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	data, err := json.Marshal(balance)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func Withdraw(db repository.Repository, w http.ResponseWriter, r *http.Request) {
	var wdr repository.WithdrawRequest
	err := ReadRequestData(r, &wdr)
	if err != nil {
		MakeErrResponse(&w, http.StatusBadRequest, "неверный формат запроса")
		return
	}

	number, err := strconv.Atoi(wdr.Order)
	if err != nil {
		MakeErrResponse(&w, http.StatusBadRequest, err.Error())
	}
	if !luhn.Valid(number) {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}
	balance, err := db.GetUserBalance(getUserId(r))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if balance.Current < wdr.Sum {
		MakeErrResponse(&w, http.StatusPaymentRequired, "на счету недостаточно средств")
		return
	}
	orderNum := strconv.Itoa(number)
	code, err := db.MakeWithdraw(orderNum, wdr.Sum, getUserId(r))
	if err != nil {
		MakeErrResponse(&w, code, err.Error())
		return
	}

	w.WriteHeader(http.StatusOK)
}

func Withdraws(db repository.Repository, w http.ResponseWriter, r *http.Request) {
	wdrws, err := db.GetUserWithdrawals(getUserId(r))
	if err != nil {
		MakeErrResponse(&w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(wdrws) == 0 {
		MakeErrResponse(&w, http.StatusNoContent, "нет ни одного списания")
		return
	}

	res, marshErr := json.Marshal(wdrws)
	if marshErr == nil {
		w.Header().Set("content-type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(res)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
