package main

import (
    "fmt"
    "net"
    "os"
)

func main() {
    // 监听 1080 端口
    listener, err := net.Listen("tcp", "0.0.0.0:8080")
    if err != nil {
        fmt.Println("Error listening:", err.Error())
        os.Exit(1)
    }
    fmt.Println("Listening on 0.0.0.0:8080 ...")
    defer listener.Close()

    // 处理连接请求
    for {
        conn, err := listener.Accept()
        if err != nil {
            fmt.Println("Error accepting:", err.Error())
            continue
        }
        go handleConnection(conn)
    }
}

func handleConnection(conn net.Conn) {
    // 处理连接请求
    defer conn.Close()

    // 接收客户端发送的 Socket5 协议版本号
    buf := make([]byte, 257)
    _, err := conn.Read(buf)
    if err != nil {
        fmt.Println("Error reading version:", err.Error())
        return
    }

    // 检查 Socket5 协议版本号
    if buf[0] != 0x05 {
        fmt.Println("Invalid protocol version:", buf[0])
        return
    }

    // 响应客户端，告知支持的 Socket5 协议版本号和支持的身份验证方式
    auth := []byte{0x05, 0x00}
    conn.Write(auth)

    // 接收客户端发送的请求类型
    n, err := conn.Read(buf)
    if err != nil {
        fmt.Println("Error reading request type:", err.Error())
        return
    }

    // 判断请求类型
    if buf[1] != 0x01 {
        fmt.Println("Unsupported request type:", buf[1])
        return
    }

    // 解析请求的目标地址
    destAddr := ""
    switch buf[3] {
    case 0x01:
        // IPv4 地址
        destAddr = fmt.Sprintf("%d.%d.%d.%d:%d", buf[4], buf[5], buf[6], buf[7], (uint16(buf[8])<<8)|uint16(buf[9]))
    case 0x03:
        // 域名
        destAddr = string(buf[5 : 5+buf[4]]) + fmt.Sprintf(":%d", (uint16(buf[5+buf[4]])<<8)|uint16(buf[5+buf[4]+1]))
    default:
        fmt.Println("Unsupported address type:", buf[3])
        return
    }

    // 建立目标服务器的连接，并向客户端发送响应
    destConn, err := net.Dial("tcp", destAddr)
    if err != nil {
        fmt.Println("Error connecting to server:", err.Error())
        return
    }
    defer destConn.Close()

    resp := []byte{0x05, 0x00, 0x00, 0x01}
    resp = append(resp, destConn.LocalAddr().(*net.TCPAddr).IP.To4()...)
    resp = append(resp, byte(destConn.LocalAddr().(*net.TCPAddr).Port>>8), byte(destConn.LocalAddr().(*net.TCPAddr).Port))
    conn.Write(resp)

    // 在客户端和服务器之间进行数据转发，直到连接关闭
    go func() {
        // 从目标服务器读取数据并转发给客户端
        buf := make([]byte, 1024)
        for {
            n, err := destConn.Read(buf)
            if err != nil {
                fmt.Println("Error reading from server:", err.Error())
                break
            }
            conn.Write(buf[:n])
        }
    }()

    // 从客户端读取数据并转发给目标服务器
    buf = make([]byte, 1024)
    for {
        n, err = conn.Read(buf)
        if err != nil {
            fmt.Println("Error reading from client:", err.Error())
            break
        }
        destConn.Write(buf[:n])
    }
}

