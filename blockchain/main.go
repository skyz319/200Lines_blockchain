package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

// Data model
type Block struct {
	Index     int    //	区块链数据记录位置
	Timestamp string //	数据写入时间戳
	BPM       int    //	数据
	Hash      string //	此数据的SHA256标识
	PrevHash  string //	前记录的SHA25标识
}

type Message struct {
	BPM int
}

var Blockchain []Block

/*
	创建Block数据的HASH
*/
func calculateHash(block Block) string {

	record := string(block.Index) + block.Timestamp + string(block.BPM) + block.PrevHash
	h := sha256.New()
	h.Write([]byte(record))
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed)
}

/*
	生成新区块
*/
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

//	网络服务
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

// GET 处理
func handleGetBlockchain(w http.ResponseWriter, r *http.Request) {

	bytes, err := json.MarshalIndent(Blockchain, "", " ")

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	io.WriteString(w, string(bytes))
}

// POST处理
func handleWriteBlock(w http.ResponseWriter, r *http.Request) {

	var m Message

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&m); err != nil {
		respondWithJSON(w, r, http.StatusBadRequest, m)
		return
	}

	defer r.Body.Close()

	//	生成新区块并写入数据
	newBlock, err := generateBlock(Blockchain[len(Blockchain)-1], m.BPM)
	if err != nil {
		respondWithJSON(w, r, http.StatusInternalServerError, m)
		return
	}

	//	验证区块是否有效
	if isBlockValid(newBlock, Blockchain[len(Blockchain)-1]) {

		newBlockchain := append(Blockchain, newBlock)
		replaceChain(newBlockchain)
		spew.Dump(Blockchain)
	}

	//	操作成功 显示写入的区块
	respondWithJSON(w, r, http.StatusCreated, newBlock)
}

func respondWithJSON(w http.ResponseWriter, r *http.Request, code int, payload interface{}) {

	response, err := json.MarshalIndent(payload, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("HTTP 500: Internal Server Error"))
		return
	}

	w.WriteHeader(code)
	w.Write(response)
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		t := time.Now()
		//	创世区块
		genesisBlock := Block{0, t.String(), 0, "", "Genesis!"}
		hash := calculateHash(genesisBlock) //	生成创世区块Hash
		genesisBlock.Hash = hash
		spew.Dump(genesisBlock)
		//	上链
		Blockchain = append(Blockchain, genesisBlock)
	}()

	log.Fatal(run())
}
