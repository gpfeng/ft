## description
file transfer includes a server and a client
file server listen on a tcp port(default 12345) and wait for client to connect
client connect to the server and send file(s) to the server

# features
permission of file received  will be kept the same with client

# build
go build ft_client.go
go build ft_client.go

# usage
./ft_server [-h hostname/ip] [-p port] [-d target directory]
./ft_client [-h hostname/ip] [-p port] file1 [file2 ...]
