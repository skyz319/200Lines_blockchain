package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/davecgh/go-spew/spew"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

type Block struct {
	Index     int
	Timestamp string
	BPM       int
	Hash      string
	PrevHash  string
}

var Blockchain []Block

// bcServer handles incoming concurrent Blocks
var bcServer chan []Block

func calculateHash(block Block) string {

	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

func generateBlock(oldBlock Block, BPM int) (Block, error) {

	var newBlock Block

	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.BPM = BPM
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Hash = calculateHash(newBlock)

	return newBlock, nil
}

func isBlockValid(newBlock, oldBlock Block) bool {

	if oldBlock.Index+1 != newBlock.Index {

		return false
	}

	if oldBlock.Hash != newBlock.PrevHash {

		return false
	}
	hashed := calculateHash(newBlock)
	if hashed != newBlock.Hash {

		return false
	}

	return true
}

func replaceChain(newBlockchain []Block) {

	if len(newBlockchain) > len(Blockchain) {
		Blockchain = newBlockchain
	}
}

func main() {

	err := godotenv.Load()
	if err != nil {

		log.Fatalln(err)
	}

	bcServer = make(chan []Block)

	//	生成创世区块
	t := time.Now()
	genesisBlock := Block{0, t.String(), 0, "", "Genesis!"}
	hashed := calculateHash(genesisBlock)
	genesisBlock.Hash = hashed
	spew.Dump(genesisBlock)
	//	上链
	Blockchain = append(Blockchain, genesisBlock)

	//	TCP Server
	server, err := net.Listen("tcp", ":"+os.Getenv("ADDR"))
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Listening Port:", os.Getenv("ADDR"))

	defer server.Close()

	//	开始接受新的连接
	for {

		conn, err := server.Accept()
		if err != nil {

			log.Fatal(err)
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {

	defer conn.Close()

	//	提示输入数据
	io.WriteString(conn, "Enter a new BPM:")
	scanner := bufio.NewScanner(conn)

	//	获取数据并上链
	go func() {

		for scanner.Scan() {

			bpm, err := strconv.Atoi(scanner.Text())
			if err != nil {

				log.Printf("%v not a number: %v", scanner.Text(), err)
				continue
			}

			newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], bpm)
			if err != nil {

				log.Println(err)
				continue
			}

			if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {

				newBlockchain := append(Blockchain, newBlock)
				replaceChain(newBlockchain)

			}

			bcServer <- Blockchain

			io.WriteString(conn, "\n Enter a new BPM:")
		}
	}()

	go func() {

		for {
			time.Sleep(10 * time.Second)
			output, err := json.Marshal(Blockchain)
			if err != nil {
				log.Fatal(err)
			}
			io.WriteString(conn, string(output))
		}
	}()

	for _ = range bcServer {
		spew.Dump(Blockchain)
	}
}
