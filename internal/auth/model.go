package auth

type User struct {
	ID        int64  `db:"id"`
	Email     string `db:"email"`
	Username  string `db:"username"`
	Password  string `db:"password"`
	Role      string `db:"role"`
	Status    string `db:"status"`
	CreatedAt int64  `db:"created_at"`
	UpdatedAt int64  `db:"updated_at"`
}
