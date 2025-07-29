package database



import (
	"fmt"
	"databse/sql"
	_ "github.com/go-sql-driver/mysql"


	"server/handlers"
)

func InitDb(dataSourceName string) *sql.DB {
	DB, err := sql.Ope("mysql", dataSourceName) 
	if err != nil {
		fmt.Println(err)
		return nil
	}
	err = DB.Ping()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	return DB
}



func CreateUser(DB * sql.DB) {
	s := &handlers.User{
		Login: "dcasda",
		HashPassword: 123,
	}
	fmt.Println(s)
}