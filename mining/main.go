package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

//	需要的HASH前导0的数量
const difficulty = 1

type Block struct {
	Index      int
	Timestamp  string
	BPM        int
	Hash       string
	PrevHash   string
	Difficulty int
	Nonce      string
}

var Blockchain []Block

type Message struct {
	BPM int
}

var mutex = &sync.Mutex{}

func run() error {

	mux := makeMuxRouter()
	httpAddr := os.Getenv("ADDR")
	log.Println("Listening on:", os.Getenv("ADDR"))

	s := &http.Server{
		Addr:           ":" + httpAddr,
		Handler:        mux,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	if err := s.ListenAndServe(); err != nil {

		return err
	}

	return nil
}

func makeMuxRouter() http.Handler {

	muxRouter := mux.NewRouter()
	muxRouter.HandleFunc("/", handleGetBlockchain).Methods("GET")
	muxRouter.HandleFunc("/", handleWriteBlock).Methods("POST")

	return muxRouter
}

//	GET handle
func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {

	bytes, err := json.MarshalIndent(Blockchain, "", " ")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	io.WriteString(w, string(bytes))
}

// POST handle
func handleWriteBlock(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Content-Type", "application/json")
	var m Message

	decode := json.NewDecoder(r.Body)
	if err := decode.Decode(&m); err != nil {
		respondWithJson(w, r, http.StatusBadRequest, r.Body)
		return
	}
	defer r.Body.Close()

	//	生成新区块时保证其原子性
	mutex.Lock()
	newBlock := generateBlock(Blockchain[len(Blockchain)-1], m.BPM)
	mutex.Unlock()

	//	验证区块
	if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {

		Blockchain = append(Blockchain, newBlock)
		spew.Dump(Blockchain)
	}

	respondWithJson(w, r, http.StatusCreated, newBlock)
}

func respondWithJson(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {

	response, err := json.MarshalIndent(payload, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: Internal Server Error"))
		return
	}

	w.WriteHeader(code)
	w.Write(response)
}

/*
	生成新区块
*/
func generateBlock(oldBlock Block, BPM int) Block {

	var newBlock Block

	t := time.Now()

	newBlock.Index = oldBlock.Index + 1
	newBlock.Timestamp = t.String()
	newBlock.BPM = BPM
	newBlock.PrevHash = oldBlock.Hash
	newBlock.Difficulty = difficulty

	//	计算HASH
	for i := 0; ; i++ {

		hex := fmt.Sprintf("%x", i)
		newBlock.Nonce = hex

		if !isHashValid(calculateHash(newBlock), newBlock.Difficulty) {

			fmt.Println(calculateHash(newBlock), "do more work!")
			time.Sleep(time.Second)
			continue
		} else {

			fmt.Println("Total:", i, "  ", calculateHash(newBlock), "work done!")
			newBlock.Hash = calculateHash(newBlock)
			break
		}
	}

	return newBlock
}

/*
	创建Block数据的HASH
*/
func calculateHash(block Block) string {

	record := strconv.Itoa(block.Index) + block.Timestamp + strconv.Itoa(block.BPM) + block.PrevHash + block.Nonce
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

/*
	块验证
*/
func isBlockValid(newBlock, oldBlock Block) bool {

	if oldBlock.Index+1 != newBlock.Index {
		return false
	}

	if oldBlock.Hash != newBlock.PrevHash {
		return false
	}

	if calculateHash(newBlock) != newBlock.Hash {
		return false
	}

	return true
}

/*
	链合并，以最长链为有效链
*/

func replaceChain(newBlockchain []Block) {

	if len(newBlockchain) > len(Blockchain) {
		Blockchain = newBlockchain
	}
}

// Proof of Work
// 检查前导0的个数
func isHashValid(hash string, difficulty int) bool {

	prefix := strings.Repeat("0", difficulty)
	return strings.HasPrefix(hash, prefix)
}

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	go func() {

		t := time.Now()
		//	创世区块
		genesisBlock := Block{}
		genesisBlock = Block{0, t.String(), 0, calculateHash(genesisBlock), "", difficulty, "GenesisBlock!"}
		spew.Dump(genesisBlock)

		mutex.Lock()
		Blockchain = append(Blockchain, genesisBlock)
		mutex.Unlock()
	}()

	log.Fatal(run())
}
