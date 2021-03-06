package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"srnd"
	"strings"
	"syscall"
)

func main() {

	daemon := new(srnd.NNTPDaemon)
	if len(os.Args) > 1 {
		action := os.Args[1]
		if action == "setup" {
			log.Println("Setting up SRNd base...")
			daemon.Setup()
			log.Println("Setup Done")
		} else if action == "run" {
			log.Printf("Starting up %s...", srnd.Version())
			daemon.Setup()
			c := make(chan os.Signal, 1)
			signal.Notify(c, os.Interrupt)
			signal.Notify(c, syscall.SIGTERM, syscall.SIGHUP)
			go func() {
				for {
					sig := <-c
					if sig != syscall.SIGHUP {
						break
					}
					srnd.ReloadTemplates()
					daemon.Reload()
				}
				log.Println("Shutting down...")
				daemon.End()
				os.Exit(0)
			}()
			daemon.Run()
		} else if action == "tool" {
			if len(os.Args) > 2 {
				tool := os.Args[2]
				if tool == "mod" {
					if len(os.Args) >= 4 && strings.ToLower(os.Args[3]) == "do" {
						daemon.Setup()
						eng := daemon.ModEngine()
						r := bufio.NewReader(os.Stdin)
						var err error
						for err == nil {
							var line string
							fmt.Print("\nmod> ")
							line, err = r.ReadString(10)
							if err == nil {
								line = strings.Trim(line, "\n")
								if line == "help" {
									fmt.Println("usage:")
									fmt.Println("delete <message-id>")
									fmt.Println("overchan-inet-ban 1.1.1.1/32")
									fmt.Println("exit")
								} else if line == "quit" || line == "exit" {
									fmt.Println("bai")
									return
								} else {
									ev := srnd.ParseModEvent(line)
									eng.Do(ev)
								}
							}
						}
					} else if len(os.Args) >= 5 {
						action := os.Args[3]
						if action == "add" {
							pk := os.Args[4]
							daemon.Setup()
							db := daemon.GetDatabase()
							err := db.MarkModPubkeyGlobal(pk)
							if err != nil {
								log.Fatal(err)
							}
						} else if action == "del" {
							pk := os.Args[4]
							daemon.Setup()
							db := daemon.GetDatabase()
							err := db.UnMarkModPubkeyGlobal(pk)
							if err != nil {
								log.Fatal(err)
							}
						} else {
							fmt.Fprintf(os.Stdout, "usage: %s tool mod [[add|del] pubkey]|[do modactiongoeshere]\n", os.Args[0])
						}
					} else {
						fmt.Fprintf(os.Stdout, "usage: %s tool mod [[add|del] pubkey]|[do modactiongoeshere]\n", os.Args[0])
					}
				} else if tool == "expire" {
					daemon.Setup()
					daemon.ExpireAll()
				} else if tool == "rethumb" {
					if len(os.Args) >= 4 {
						threads := runtime.NumCPU()
						arg := strings.ToLower(os.Args[3])
						switch arg {
						case "missing":
							srnd.ThumbnailTool(threads, true)
							return
						case "all":
							srnd.ThumbnailTool(threads, false)
							return
						}
					}
					fmt.Fprintf(os.Stdout, "usage: %s tool rethumb [missing|all]\n", os.Args[0])
				} else if tool == "keygen" {
					srnd.KeygenTool()
				} else if tool == "nntp" {
					if len(os.Args) >= 5 {
						action := os.Args[3]
						if action == "del-login" {
							daemon.Setup()
							daemon.DelNNTPLogin(os.Args[4])
						} else if action == "add-login" {
							if len(os.Args) == 6 {
								username := os.Args[4]
								passwd := os.Args[5]
								daemon.Setup()
								daemon.AddNNTPLogin(username, passwd)
							} else {
								fmt.Fprintf(os.Stdout, "Usage: %s tool nntp add-login username password\n", os.Args[0])
							}
						} else {
							fmt.Fprintf(os.Stdout, "Usage: %s tool nntp [add-login|del-login]\n", os.Args[0])
						}
					} else {
						fmt.Fprintf(os.Stdout, "Usage: %s tool nntp [add-login|del-login]\n", os.Args[0])
					}
				} else {
					fmt.Fprintf(os.Stdout, "Usage: %s tool [rethumb|keygen|nntp|mod|expire]\n", os.Args[0])
				}
			} else {
				fmt.Fprintf(os.Stdout, "Usage: %s tool [rethumb|keygen|nntp|mod|expire]\n", os.Args[0])
			}
		} else {
			log.Println("Invalid action:", action)
		}
	} else {
		fmt.Fprintf(os.Stdout, "Usage: %s [setup|run|tool]\n", os.Args[0])
	}
}
