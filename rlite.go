package main

import (
    "bufio"
    "bytes"
    "fmt"
    "log"
    "net"
    "os"
    "os/exec"
    "runtime"
    "runtime/debug"
    "strings"
)

func checkPassword(user, pass string) bool {
    cmd := exec.Command("login", "-f", user)
    cmd.Stdin = bytes.NewBufferString(pass + "\n")
    err := cmd.Run()
    return err == nil
}

func isBlocked(cmdLine string) bool {
    clean := strings.ToLower(strings.TrimSpace(cmdLine))

    // Detect fork bombs (cualquier variación)
    if strings.Contains(clean, ":(){") && strings.Contains(clean, "|:&") {
        return true
    }

    dangerousContains := []string{
        "rm -rf /",
        "rm --no-preserve-root",
        "mkfs",
        "mkfs.ext4",
        "mkfs.fat",
        "mkfs.ntfs",
        "/dev/sd",
        "dd if=",
        "shutdown",
        "reboot",
        "halt",
        "poweroff",
        "iptables",
        "nft ",
        "useradd",
        "userdel",
        "passwd",
    }

    for _, d := range dangerousContains {
        if strings.Contains(clean, d) {
            return true
        }
    }

    return false
}

func main() {
    // Limit to 1 CPU thread
    runtime.GOMAXPROCS(1)

    // Limit RAM to 256 MB (change if needed)
    debug.SetMemoryLimit(256 * 1024 * 1024)

    port := ":2121"
    if os.Geteuid() == 0 {
        port = ":21"
    }

    fmt.Println("R-Lite running on", port)

    ln, err := net.Listen("tcp", port)
    if err != nil {
        log.Fatal("Error starting R-Lite:", err)
    }

    for {
        conn, err := ln.Accept()
        if err == nil {
            go handle(conn)
        }
    }
}

func handle(conn net.Conn) {
    defer conn.Close()
    reader := bufio.NewReader(conn)

    // Colors
    cyan := "\033[36m"
    green := "\033[32m"
    red := "\033[31m"
    reset := "\033[0m"

    // Banner
    conn.Write([]byte(cyan + "=== Welcome to R-Lite ===\n" + reset))
    conn.Write([]byte("User: "))

    user, _ := reader.ReadString('\n')
    user = strings.TrimSpace(user)

    conn.Write([]byte("Password: "))
    pass, _ := reader.ReadString('\n')
    pass = strings.TrimSpace(pass)

    if !checkPassword(user, pass) {
        conn.Write([]byte(red + "Access denied\n" + reset))
        return
    }

    conn.Write([]byte(green + "Access granted\n\n" + reset))

    for {
        // Prompt with username
        conn.Write([]byte(green + user + "> " + reset))

        cmdLine, _ := reader.ReadString('\n')
        cmdLine = strings.TrimSpace(cmdLine)

        if cmdLine == "exit" {
            conn.Write([]byte("Closing session...\n"))
            return
        }

        if cmdLine == "" {
            continue
        }

        parts := strings.Fields(cmdLine)
        cmd := exec.Command(parts[0], parts[1:]...)

        out, err := cmd.CombinedOutput()
        if err != nil {
            conn.Write([]byte(red + "Error: " + err.Error() + "\n" + reset))
        }

        conn.Write(out)
    }
}
