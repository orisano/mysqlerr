package mysqlerr

//go:generate go run ./cmd/mysqlerrgen -pkg mysqlerr8 -url https://raw.githubusercontent.com/mysql/mysql-server/mysql-8.1.0/share/messages_to_clients.txt
//go:generate go run ./cmd/mysqlerrgen -pkg mysqlerr80 -url https://raw.githubusercontent.com/mysql/mysql-server/mysql-8.0.33/share/messages_to_clients.txt
//go:generate go run ./cmd/mysqlerrgen -pkg mysqlerr57 -url https://raw.githubusercontent.com/mysql/mysql-server/mysql-5.7.42/sql/share/errmsg-utf8.txt

