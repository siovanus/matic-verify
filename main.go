package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

const BorRPCUrl = "http://43.133.170.42:8545"

func jsonRequest(url string, data []byte) (result []byte, err error) {
	resp, err := http.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return
	}

	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

type HeaderWithOptionalSnap struct {
	Header   ethtypes.Header
	Snapshot *Snapshot
}
type Snapshot struct {
	Hash         ethcommon.Hash `json:"hash"`         // Block hash where the snapshot was created
	ValidatorSet *ValidatorSet  `json:"validatorSet"` // Validator set at this moment
}

type ValidatorSet struct {
	// NOTE: persisted via reflect, must be exported.
	Validators []*Validator `json:"validators"`
	Proposer   *Validator   `json:"proposer"`

	// cached (unexported)
	totalVotingPower int64
}

type Validator struct {
	ID               uint64            `json:"ID"`
	Address          ethcommon.Address `json:"signer"`
	VotingPower      int64             `json:"power"`
	ProposerPriority int64             `json:"accum"`
}

func getBorGenesis() {
	type heightReq struct {
		JsonRpc string   `json:"jsonrpc"`
		Method  string   `json:"method"`
		Params  []string `json:"params"`
		Id      uint     `json:"id"`
	}
	req1 := &heightReq{
		JsonRpc: "2.0",
		Method:  "eth_blockNumber",
		Params:  make([]string, 0),
		Id:      1,
	}
	data, _ := json.Marshal(req1)

	body, err := jsonRequest(BorRPCUrl, data)
	if err != nil {
		return
	}
	var resp1 struct {
		JSONRPC string `json:"jsonrpc"`
		Result  string `json:"result"`
		ID      uint   `json:"id"`
	}
	err = json.Unmarshal(body, &resp1)
	if err != nil {
		return
	}
	height, err := strconv.ParseUint(resp1.Result, 0, 64)
	if err != nil {
		return
	}

	type BlockReq struct {
		JsonRpc string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
		Id      uint          `json:"id"`
	}
	req2 := &BlockReq{
		JsonRpc: "2.0",
		Method:  "eth_getBlockByNumber",
		Params:  []interface{}{fmt.Sprintf("0x%x", height), true},
		Id:      1,
	}
	data, _ = json.Marshal(req2)

	body, err = jsonRequest(BorRPCUrl, data)
	if err != nil {
		return
	}
	var resp struct {
		JSONRPC string           `json:"jsonrpc"`
		Result  *ethtypes.Header `json:"result"`
		ID      uint             `json:"id"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return
	}
	borHeader := resp.Result

	snapshot, err := borGetSnapshotAtHash(borHeader.Hash())
	if err != nil {
		return
	}
	genesis := &HeaderWithOptionalSnap{
		Header:   *borHeader,
		Snapshot: snapshot,
	}
	raw, _ := json.Marshal(genesis)
	fmt.Println(string(raw))
	fmt.Println("bor genesis: ", hex.EncodeToString(raw))
}

func borGetSnapshotAtHash(hash ethcommon.Hash) (*Snapshot, error) {
	req := struct {
		JSONRPC string
		Method  string
		Params  []string
		ID      int
	}{
		JSONRPC: "2.0",
		Method:  "bor_getSnapshotAtHash",
		Params:  []string{hash.Hex()},
		ID:      1,
	}

	data, _ := json.Marshal(req)

	body, err := jsonRequest(BorRPCUrl, data)
	if err != nil {
		return nil, err
	}

	var resp struct {
		JSONRPC string    `json:"jsonrpc"`
		Result  *Snapshot `json:"result"`
		ID      uint      `json:"id"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	return resp.Result, err
}

func main() {
	getBorGenesis()
	return
}
