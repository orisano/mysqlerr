package mysqlerr

//go:generate go run ./cmd/mysqlerrgen -pkg mysqlerr8 -url https://raw.githubusercontent.com/mysql/mysql-server/mysql-8.4.7/share/messages_to_clients.txt
//go:generate go run ./cmd/mysqlerrgen -pkg mysqlerr80 -url https://raw.githubusercontent.com/mysql/mysql-server/mysql-8.0.44/share/messages_to_clients.txt
//go:generate go run ./cmd/mysqlerrgen -pkg mysqlerr57 -url https://raw.githubusercontent.com/mysql/mysql-server/mysql-5.7.44/sql/share/errmsg-utf8.txt
//go:generate go run ./cmd/mysqlerrgen -pkg mysqlerr9 -url https://raw.githubusercontent.com/mysql/mysql-server/mysql-9.7.2/share/messages_to_clients.txt
//go:generate go run ./cmd/mysqlerrgen -pkg mysqlerr26 -url https://raw.githubusercontent.com/mysql/mysql-server/mysql-26.7.0/share/messages_to_clients.txt

