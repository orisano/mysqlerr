package mysqlerr

//go:generate go run ./cmd/mysqlerrgen -pkg mysqlerr80 -url https://raw.githubusercontent.com/mysql/mysql-server/mysql-8.0.31/share/messages_to_clients.txt
//go:generate go run ./cmd/mysqlerrgen -pkg mysqlerr57 -url https://raw.githubusercontent.com/mysql/mysql-server/mysql-5.7.40/sql/share/errmsg-utf8.txt

