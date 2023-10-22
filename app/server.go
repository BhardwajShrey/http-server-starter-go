package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
)

const (
    OK = "HTTP/1.1 200 OK\r\n\r\n"
    CREATED = "HTTP/1.1 201 CREATED\r\n\r\n"
    NOT_FOUND = "HTTP/1.1 404 NOT FOUND\r\n\r\n"
)

func plainTextResponse(s string) string {
    return fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(s), s)
}

func getFileResponse(contents string) string {
    return fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Type: application/octet-stream\r\nContent-Length: %d\r\n\r\n%s", len(contents), contents)
}

func readFile(dirname string, filepath string) (string, error) {
    contents, err := os.ReadFile(dirname + "/" + filepath)
    return string(contents), err
}

func writeFile(dirname, filepath, contents string) error {
    return os.WriteFile(dirname + "/" + filepath, []byte(contents), 0666)
}

func main() {
    dir := flag.String("directory", "", "directory")
    flag.Parse()

    l, err := net.Listen("tcp", "0.0.0.0:4221")
    if err != nil {
        fmt.Println("Failed to bind to port 4221")
        os.Exit(1)
    }

    for {
        connection, err := l.Accept()
        if err != nil {
            fmt.Println("Error accepting connection: ", err.Error())
            os.Exit(1)
        }

        go handleConnection(connection, dir)
    }
}

func handleConnection(connection net.Conn, dir *string) {
    defer connection.Close()

    buffer := make([]byte, 1024)
    _, err := connection.Read(buffer)
    if err != nil {
        fmt.Println("Error reading buffer from connection: ", err.Error())
        os.Exit(1)
    }

    req := string(buffer)
    reqArray := strings.Split(req, "\r\n")
    reqInfo := strings.Split(reqArray[0], " ")
    reqType := reqInfo[0]
    path := reqInfo[1]

    var res string
    switch {
    case path == "/":
        res = OK
    case strings.HasPrefix(path, "/echo"):
        pathSplit := strings.Split(path, "/")
        param := strings.Join(pathSplit[2:], "/")
        res = plainTextResponse(param)
    case strings.HasPrefix(path, "/user-agent"):
        userAgent := strings.Split(reqArray[2], " ")[1]
        res = plainTextResponse(userAgent)
    case strings.HasPrefix(path, "/files"):
        filepath := strings.Split(path, "/")[2]
        switch reqType {
        case "POST":
            fileContents := strings.Split(req, "\r\n\r\n")[1]
            err := writeFile(*dir, filepath, fileContents)
            c, _ := readFile(*dir, filepath)
            if c == fileContents {
                fmt.Println("Contents matching...")
                fmt.Printf("Expected: %v\nActual: %v\n", fileContents, c)
            } else {
                fmt.Printf("Expected: %v\nActual: %v\n", fileContents, c)
            }
            if err != nil {
                fmt.Println("Failed to write to file.")
            } else {
                res = CREATED
            }
        default:
            // GET
            contents, err := readFile(*dir, filepath)
            if err != nil {
                res = NOT_FOUND
            } else {
                res = getFileResponse(contents)
            }
        }
    default:
        res = NOT_FOUND
    }

    _, err = connection.Write([]byte(res))
    if err != nil {
        fmt.Println("Error sending response: ", err.Error())
        os.Exit(1)
    }
}
