package repository

import (
	"database/sql"
	"errors"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/kuznet1/gophermart/internal/errs"
	"github.com/kuznet1/gophermart/internal/model"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 14

type Repo struct {
	db *sql.DB
}

func NewRepo(db *sql.DB) *Repo {
	return &Repo{
		db: db,
	}
}

func (r *Repo) Register(user model.UserCredentials) (int, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcryptCost)
	if err != nil {
		return 0, err
	}
	var userID int
	err = r.db.QueryRow("INSERT INTO users (login, password) VALUES ($1, $2) RETURNING id", user.Login, passwordHash).Scan(&userID)
	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation {
		return 0, errs.ErrUserExists
	}
	if err != nil {
		return 0, err
	}
	return userID, nil
}

func (r *Repo) Login(user model.UserCredentials) (int, error) {
	var userID int
	var passHash string
	row := r.db.QueryRow("SELECT id, password  FROM users WHERE login = $1", user.Login)
	if err := row.Scan(&userID, &passHash); err != nil {
		return 0, errs.ErrUserCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(passHash), []byte(user.Password)); err != nil {
		return 0, errs.ErrUserCredentials
	}
	return userID, nil
}

func (r *Repo) AddOrder(userID int, orderNum int) error {
	query := "INSERT INTO orders (order_id, user_id) VALUES ($1, $2)"
	_, err := r.db.Exec(query, orderNum, userID)
	var e *pgconn.PgError
	if errors.As(err, &e) && e.Code == pgerrcode.UniqueViolation {
		var id int
		query = "SELECT user_id FROM orders WHERE order_id = $1"
		if err = r.db.QueryRow(query, orderNum).Scan(&id); err != nil {
			return err
		}
		if userID != id {
			return errs.ErrOrderUploadedByOtherUser
		} else {
			return errs.ErrOrderUploadedByUser
		}
	}
	return err
}

func (r *Repo) GetOrders(userID int) ([]model.Order, error) {
	query := "SELECT order_id, status, accrual, uploaded_at FROM orders WHERE user_id = $1  ORDER BY uploaded_at DESC"
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orders := make([]model.Order, 0)

	for rows.Next() {
		var order model.Order
		err = rows.Scan(&order.Order, &order.Status, &order.Accrual, &order.UploadedAt)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *Repo) NewWithdrawal(userID int, withdraws model.Withdraw) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}

	done := false
	defer func() {
		if done {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	balance, err := r.doGetBalance(tx, userID)
	if err != nil {
		return err
	}

	if withdraws.Sum > balance.Current {
		return errs.ErrBalanceNotEnoughPoints
	}

	query := "INSERT INTO withdrawals (user_id, order_id, sum) VALUES ($1, $2, $3)"
	_, err = r.db.Exec(query, userID, withdraws.Order, withdraws.Sum)
	if err != nil {
		return err
	}
	done = true
	return nil
}

func (r *Repo) GetWithdrawals(userID int) ([]model.Withdrawal, error) {
	query := "SELECT order_id, sum, processed_at FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC"
	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	withdrawals := make([]model.Withdrawal, 0)
	for rows.Next() {
		var withdrawal model.Withdrawal
		err = rows.Scan(&withdrawal.Order, &withdrawal.Sum, &withdrawal.ProcessedAt)
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return withdrawals, nil
}

func (r *Repo) GetBalance(userID int) (model.Balance, error) {
	tx, err := r.db.Begin()
	if err != nil {
		return model.Balance{}, err
	}

	done := false
	defer func() {
		if done {
			tx.Commit()
		} else {
			tx.Rollback()
		}
	}()

	balance, err := r.doGetBalance(tx, userID)
	if err != nil {
		return model.Balance{}, err
	}
	done = true
	return balance, nil
}

func (r *Repo) UpdateAccrual(accrual model.AccrualResp) error {
	_, err := r.db.Exec(
		"UPDATE orders SET status=$1, accrual=$2 WHERE order_id = $3",
		accrual.Status, accrual.Accrual, accrual.Order,
	)
	return err
}

func (r *Repo) GetProcessingOrders() ([]int, error) {
	query := "SELECT order_id FROM orders WHERE status IN ('NEW', 'PROCESSING')"
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orders []int

	for rows.Next() {
		var order int
		err = rows.Scan(&order)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *Repo) doGetBalance(tx *sql.Tx, userID int) (model.Balance, error) {
	row := tx.QueryRow("SELECT coalesce(SUM(accrual), 0) FROM orders WHERE user_id = $1", userID)
	var sumAccruals float64
	if err := row.Scan(&sumAccruals); err != nil {
		return model.Balance{}, err
	}

	row = tx.QueryRow("SELECT coalesce(SUM(sum), 0) FROM withdrawals WHERE user_id = $1", userID)
	var sumWithdrawals float64
	if err := row.Scan(&sumWithdrawals); err != nil {
		return model.Balance{}, err
	}

	return model.Balance{
		Current:   sumAccruals - sumWithdrawals,
		Withdrawn: sumWithdrawals,
	}, nil
}
