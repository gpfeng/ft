# description
file transfer includes a server and a client<br />
file server listen on a tcp port(default 12345) and wait for client to connect<br />
client connect to the server and send file(s) to the server<br />

## features
permission of file received  will be kept the same with client<br />

## build
go build ft_client.go<br />
go build ft_client.go<br />

## usage
./ft_server [-h hostname/ip] [-p port] [-d target directory]<br />
./ft_client [-h hostname/ip] [-p port] file1 [file2 ...]<br />
