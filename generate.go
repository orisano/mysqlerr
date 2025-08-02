package mysqlerr

//go:generate go run ./cmd/mysqlerrgen -pkg mysqlerr8 -url https://raw.githubusercontent.com/mysql/mysql-server/mysql-8.4.6/share/messages_to_clients.txt
//go:generate go run ./cmd/mysqlerrgen -pkg mysqlerr80 -url https://raw.githubusercontent.com/mysql/mysql-server/mysql-8.0.43/share/messages_to_clients.txt
//go:generate go run ./cmd/mysqlerrgen -pkg mysqlerr57 -url https://raw.githubusercontent.com/mysql/mysql-server/mysql-5.7.44/sql/share/errmsg-utf8.txt

