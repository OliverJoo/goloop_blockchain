package main

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/icon-project/goloop/block"
	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/db"
	"github.com/icon-project/goloop/module"
	"github.com/icon-project/goloop/rpc"
	"github.com/icon-project/goloop/service"
)

type chain struct {
	wallet module.Wallet
	nid    int

	database db.Database
	sm       module.ServiceManager
	bm       module.BlockManager
	cs       module.Consensus
	sv       rpc.JsonRpcServer
}

func (c *chain) Database() db.Database {
	return c.database
}

func (c *chain) Wallet() module.Wallet {
	return c.wallet
}

func (c *chain) NID() int {
	return c.nid
}

func (c *chain) Genesis() []byte {
	genPath := genesisFile
	if len(genPath) == 0 {
		file := "genesisTx.json"
		topDir := "goloop"
		path, _ := filepath.Abs(".")
		base := filepath.Base(path)
		switch {
		case strings.Compare(base, topDir) == 0:
			genPath = path + "/" + file
		case strings.Compare(base, "icon-project") == 0:
			genPath = path + "/" + topDir + "/" + file
		default:
			log.Panicln("Not considered case")
		}
	}
	gen, err := ioutil.ReadFile(genPath)
	if err != nil {
		log.Panicln("Failed to read genesisFile. err : ", err)
	}
	return gen
}

func voteListDecoder([]byte) module.VoteList {
	return &emptyVoteList{}
}

func (c *chain) VoteListDecoder() module.VoteListDecoder {
	return module.VoteListDecoder(voteListDecoder)
}

type emptyVoteList struct {
}

func (vl *emptyVoteList) Verify(block module.Block, validators module.ValidatorList) error {
	return nil
}

func (vl *emptyVoteList) Bytes() []byte {
	return nil
}

func (vl *emptyVoteList) Hash() []byte {
	return crypto.SHA3Sum256(vl.Bytes())
}

type proposeOnlyConsensus struct {
	sm module.ServiceManager
	bm module.BlockManager
	ch chan<- []byte
}

func (c *proposeOnlyConsensus) Start() {
	blks, err := c.bm.FinalizeGenesisBlocks(
		common.NewAccountAddress(make([]byte, common.AddressIDBytes)),
		time.Unix(0, 0),
		&emptyVoteList{},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("Proposer: FinalizedGenesisBlocks")
	blk := blks[len(blks)-1]

	ch := make(chan module.Block)
	height := 1
	wallet := Wallet{"https://testwallet.icon.foundation/api/v3"}
	for {
		b, err := wallet.GetBlockByHeight(height)
		if err != nil {
			panic(err)
		}
		wblk, err := NewBlockV1(b)
		if err != nil {
			panic(err)
		}
		for itr := wblk.NormalTransactions().Iterator(); itr.Has(); itr.Next() {
			t, _, _ := itr.Get()
			c.sm.SendTransaction(t)
		}

		_, err = c.bm.Propose(blk.ID(), &emptyVoteList{}, func(b module.Block, e error) {
			if e != nil {
				panic(e)
			}
			ch <- b
		})
		if err != nil {
			panic(err)
		}
		blk = <-ch
		err = c.bm.Finalize(blk)
		if err != nil {
			panic(err)
		}
		buf := bytes.NewBuffer(nil)
		blk.MarshalHeader(buf)
		blk.MarshalBody(buf)
		fmt.Printf("Proposer: Finalized Block(%d) %x\n", blk.Height(), blk.ID())
		c.ch <- buf.Bytes()
		blk2, err := c.bm.GetBlockByHeight(int64(height) + 1)
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(blk.ID(), blk2.ID()) {
			panic("id not equal")
		}
		blk2, err = c.bm.GetBlock(blk.ID())
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(blk.ID(), blk2.ID()) {
			panic("id not equal")
		}
		blk2, err = c.bm.GetLastBlock()
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(blk.ID(), blk2.ID()) {
			panic("id not equal")
		}
		for it := blk2.NormalTransactions().Iterator(); it.Has(); it.Next() {
			tr, _, err := it.Get()
			if err != nil {
				panic(err)
			}
			_, err = c.bm.GetTransactionInfo(tr.ID())
			if err != nil {
				panic(err)
			}
			res, err := tr.ToJSON(2)
			if err != nil {
				panic(err)
			}
			resMap := res.(map[string]interface{})
			fmt.Printf("Proposer: tx %x\n", tr.Hash())
			txjsonHashHex := resMap["tx_hash"].(string)
			txHashHex := hex.EncodeToString(tr.Hash())
			if txjsonHashHex != txHashHex {
				panic("tx hash not equal")
			}
		}

		height++
		time.Sleep(1 * time.Second)
	}
}

func (c *proposeOnlyConsensus) GetStatus() *module.ConsensusStatus {
	return nil
}

type importOnlyConsensus struct {
	bm module.BlockManager
	sm module.ServiceManager
	ch <-chan []byte
}

func (c *importOnlyConsensus) Start() {
	_, err := c.bm.FinalizeGenesisBlocks(
		common.NewAccountAddress(make([]byte, common.AddressIDBytes)),
		time.Unix(0, 0),
		&emptyVoteList{},
	)
	if err != nil {
		panic(err)
	}
	fmt.Println("Importer : FinalizedGenesisBlocks")

	ch := make(chan module.Block)
	for {
		bs := <-c.ch
		buf := bytes.NewBuffer(bs)
		_, err := c.bm.Import(buf, func(b module.Block, e error) {
			if e != nil {
				panic(e)
			}
			ch <- b
		})
		if err != nil {
			panic(err)
		}
		blk := <-ch
		err = c.bm.Finalize(blk)
		if err != nil {
			panic(err)
		}
		fmt.Printf("Importer: Finalized Block(%d) %x\n", blk.Height(), blk.ID())
		blk2, err := c.bm.GetBlockByHeight(blk.Height())
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(blk.ID(), blk2.ID()) {
			panic("id not equal")
		}
		blk2, err = c.bm.GetBlock(blk.ID())
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(blk.ID(), blk2.ID()) {
			panic("id not equal")
		}
		blk2, err = c.bm.GetLastBlock()
		if err != nil {
			panic(err)
		}
		if !bytes.Equal(blk.ID(), blk2.ID()) {
			panic("id not equal")
		}

		b, err := c.bm.GetBlockByHeight(blk.Height())
		if err != nil {
			panic(err)
		}
		jsonMap, _ := b.ToJSON(0)
		result := jsonMap.(map[string]interface{})
		fmt.Printf("Importer: GetBlock  Block(%d) %s\n", result["height"], result["block_hash"])
	}
}

func (c *importOnlyConsensus) GetStatus() *module.ConsensusStatus {
	return nil
}

func (c *chain) startAsProposer(ch chan<- []byte) {
	c.wallet = common.NewWallet()
	c.database = db.NewMapDB()
	c.sm = service.NewManager(c)
	c.bm = block.NewManager(c, c.sm)
	c.cs = &proposeOnlyConsensus{
		sm: c.sm,
		bm: c.bm,
		ch: ch,
	}
	c.cs.Start()
}

func (c *chain) startAsImporter(ch <-chan []byte) {
	c.wallet = common.NewWallet()
	c.database = db.NewMapDB()
	c.sm = service.NewManager(c)
	c.bm = block.NewManager(c, c.sm)
	c.cs = &importOnlyConsensus{
		sm: c.sm,
		bm: c.bm,
		ch: ch,
	}

	// JSON-RPC
	c.sv = rpc.NewJsonRpcServer(c.bm, c.sm, nil)
	c.sv.Start()

	c.cs.Start()
}

type JSONRPCResponse struct {
	Version string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
}

type Wallet struct {
	url string
}

func (w *Wallet) Call(method string, params map[string]interface{}) ([]byte, error) {
	d := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if params != nil {
		d["params"] = params
	}
	req, err := json.Marshal(d)
	if err != nil {
		log.Println("Making request fails")
		log.Println("Data", d)
		return nil, err
	}
	resp, err := http.Post(w.url, "application/json", bytes.NewReader(req))
	if resp.StatusCode != 200 {
		return nil, errors.New(
			fmt.Sprintf("FAIL to call res=%d", resp.StatusCode))
	}

	var buf = make([]byte, 2048*1024)
	var bufLen, readed int = 0, 0

	for true {
		readed, _ = resp.Body.Read(buf[bufLen:])
		if readed < 1 {
			break
		}
		bufLen += readed
	}
	var r JSONRPCResponse
	err = json.Unmarshal(buf[0:bufLen], &r)
	if err != nil {
		log.Println("JSON Parse Fail")
		log.Println("JSON=", string(buf[0:bufLen]))
		return nil, err
	}
	return r.Result.MarshalJSON()
}

func (w *Wallet) GetBlockByHeight(h int) ([]byte, error) {
	p := map[string]interface{}{
		"height": fmt.Sprintf("0x%x", h),
	}
	return w.Call("icx_getBlockByHeight", p)
}

var genesisFile string

func main() {
	flag.StringVar(&genesisFile, "genesis", "", "Genesis transaction param")
	flag.Parse()
	proposer := new(chain)
	importer := new(chain)

	ch := make(chan []byte)
	go proposer.startAsProposer(ch)
	importer.startAsImporter(ch)
}
