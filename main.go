package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type Book struct {
	Author      string `json:"author"`
	ID          string `json:"id"`
	Title       string `json:"title"`
	PublishDate string `json:"publish_date"`
	ISBN        string `json:"isbn"` //international standard book number
}

type BookCheckout struct {
	BookID    string `json:"book_id"`
	User      string `json:"user"`
	CheckDate string `json:"check_date"`
	IsGenesis bool   `json:"is_genesis"`
}

type Block struct {
	Pos       int
	Data      BookCheckout
	TimeStamp string
	Hash      string
	PrevHash  string
}

// Generate hash for a block
func (block *Block) generateHash() {
	bytes, _ := json.Marshal(block.Data)
	data := string(block.Pos) + block.TimeStamp + string(bytes) + block.PrevHash

	//Creating sha256 hash using mixture of data and other properties combined like above link of code
	hash := sha256.New()
	hash.Write([]byte(data))
	block.Hash = hex.EncodeToString(hash.Sum(nil))
}

func (block *Block) validateHash(hash string) bool {
	block.generateHash()
	return block.Hash == hash

}

type BlockChain struct {
	blocks []*Block //linkedlist of blocks
}

func (b *BlockChain) AddBlock(bookCheckout BookCheckout) {
	prevBlock := b.blocks[len(b.blocks)-1] //last block

	block := CreateBlock(bookCheckout, prevBlock)

	if validBlock(block, prevBlock) {
		b.blocks = append(b.blocks, block)
	}

}

// Creating a block with chceckout data
func CreateBlock(data BookCheckout, prevBlock *Block) *Block {
	block := &Block{} //empty struct
	block.Pos = prevBlock.Pos + 1
	block.PrevHash = prevBlock.Hash
	block.Data = data
	block.TimeStamp = time.Now().String()
	block.generateHash() //new hash for current block

	return block
}

func validBlock(block *Block, prevBlock *Block) bool {
	//as whole blocks should be in  sync
	if prevBlock.Hash != block.PrevHash || !block.validateHash(block.Hash) || prevBlock.Pos+1 != block.Pos {
		return false
	}

	return true

}

// Add new Book
func newBook(rw http.ResponseWriter, r *http.Request) {
	var book Book //book struct

	//encoding and decoding is same as marshaling and unmarshaling
	if err := json.NewDecoder(r.Body).Decode(&book); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		log.Printf("Could not create:%v", err)
		rw.Write([]byte("Could not create new book"))
		return
	}
	h := md5.New() //message digest hashing ,one way hashing to authneticate the book here
	h.Write([]byte(book.ISBN + book.PublishDate))

	book.ID = fmt.Sprintf("%x", h.Sum(nil)) //returns a formatted string

	//Converting into json (encoding)
	res, err := json.MarshalIndent(book, "", " ")
	HandleError(err, rw)
	if err != nil {
		log.Printf("Could not marshal payload:%v", err)
		rw.Write([]byte("Could not save book data"))
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write(res)

}

func writeBlock(rw http.ResponseWriter, r *http.Request) {
	var bookCheckout BookCheckout
	if err := json.NewDecoder(r.Body).Decode(&bookCheckout); err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
		log.Printf("Could not write block:%v", err)
		rw.Write([]byte("Could not write block"))
		return
	}

	Blockchain.AddBlock(bookCheckout)

}

func HandleError(err error, rw http.ResponseWriter) {
	if err != nil {
		rw.WriteHeader(http.StatusInternalServerError)
	}
}

func GenesisBlock() *Block {
	return CreateBlock(BookCheckout{IsGenesis: true}, &Block{})
}
func DeployFreshBlockChain() *BlockChain {
	//create new block chain with a genesis block
	return &BlockChain{[]*Block{GenesisBlock()}}
}

func getBlockChain(w http.ResponseWriter, r *http.Request) {
	jBytes, err := json.MarshalIndent(Blockchain.blocks, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}
	io.WriteString(w, string(jBytes))

}

var Blockchain *BlockChain //All info about blockchain

func main() {

	Blockchain = DeployFreshBlockChain()
	r := mux.NewRouter()
	r.HandleFunc("/", getBlockChain).Methods("GET")
	r.HandleFunc("/", writeBlock).Methods("POST")
	r.HandleFunc("/new", newBook).Methods("POST")

	go func() {
		//as soon as program starts this should run
		for _, block := range Blockchain.blocks {
			fmt.Printf("Prev.Hash:%x\n", block.PrevHash)
			bytes, _ := json.MarshalIndent(block.Data, "", " ")
			fmt.Printf("Data:%v\n", string(bytes))
			fmt.Printf("Hash:%x\n", block.Hash)
		}
	}()

	log.Println("Listening to port 3000")
	log.Fatal(http.ListenAndServe(":3000", r))

}
