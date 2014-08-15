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
Usage: ft_server [-dhp]<br />
  -d=".": target direcotry to store received file(s)<br />
  -h=false: print help information<br />
  -p=12345: listen port<br />
Usage: ft_client [-Hchp] file1 [file2 ...]<br />
  -H="127.0.0.1": host name or ip address<br />
  -c=false: compress or not<br />
  -h=false: print help information<br />
  -p=12345: host port<br />
