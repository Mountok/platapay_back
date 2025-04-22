package models

type User struct {
	Id        int64  `json:"id" db:"id"`
	Firstname string `json:"firstname" db:"firstname"`
	Username  string `json:"username" db:"username"`
	UserId    int64  `json:"user_id" db:"user_id"`
}
