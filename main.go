package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"strconv"

	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	httpClient "github.com/tendermint/tendermint/rpc/client/http"
	"github.com/tendermint/tendermint/types"
	"github.com/zhiqiangxu/matic-verify/pkg/helper"
)

const TendermintRPCUrl = "http://tendermint.api.matic.network:80"

func VerifyCosmosHeader(myHeader *CosmosHeader, info *CosmosEpochSwitchInfo) error {
	// now verify this header
	valset := types.NewValidatorSet(myHeader.Valsets)
	if !bytes.Equal(info.NextValidatorsHash, valset.Hash()) {
		return fmt.Errorf("VerifyCosmosHeader, block validator is not right, next validator hash: %s, "+
			"validator set hash: %s", info.NextValidatorsHash.String(), hex.EncodeToString(valset.Hash()))
	}
	if !bytes.Equal(myHeader.Header.ValidatorsHash, valset.Hash()) {
		return fmt.Errorf("VerifyCosmosHeader, block validator is not right!, header validator hash: %s, "+
			"validator set hash: %s", myHeader.Header.ValidatorsHash.String(), hex.EncodeToString(valset.Hash()))
	}
	if myHeader.Commit.GetHeight() != myHeader.Header.Height {
		return fmt.Errorf("VerifyCosmosHeader, commit height is not right! commit height: %d, "+
			"header height: %d", myHeader.Commit.GetHeight(), myHeader.Header.Height)
	}
	if !bytes.Equal(myHeader.Commit.BlockID.Hash, myHeader.Header.Hash()) {
		return fmt.Errorf("VerifyCosmosHeader, commit hash is not right!, commit block hash: %s,"+
			" header hash: %s", myHeader.Commit.BlockID.Hash.String(), hex.EncodeToString(valset.Hash()))
	}
	if err := myHeader.Commit.ValidateBasic(); err != nil {
		return fmt.Errorf("VerifyCosmosHeader, commit is not right! err: %s", err.Error())
	}
	if valset.Size() != len(myHeader.Commit.Signatures) {
		return fmt.Errorf("VerifyCosmosHeader, the size of precommits is not right!")
	}
	talliedVotingPower := int64(0)
	for idx, commitSig := range myHeader.Commit.Signatures {
		if commitSig.Absent() {
			continue // OK, some precommits can be missing.
		}
		_, val := valset.GetByIndex(int32(idx))
		// Validate signature.
		precommitSignBytes := myHeader.Commit.VoteSignBytes(info.ChainID, int32(idx))
		if !val.PubKey.VerifySignature(precommitSignBytes, commitSig.Signature) {
			return fmt.Errorf("VerifyCosmosHeader, Invalid commit -- invalid signature: %v", commitSig)
		}
		// Good precommit!
		if myHeader.Commit.BlockID.Equals(commitSig.BlockID(myHeader.Commit.BlockID)) {
			talliedVotingPower += val.VotingPower
		}
	}
	if talliedVotingPower <= valset.TotalVotingPower()*2/3 {
		return fmt.Errorf("VerifyCosmosHeader, voteing power is not enough!")
	}

	return nil
}

type CosmosEpochSwitchInfo struct {
	NextValidatorsHash tmbytes.HexBytes
	ChainID            string
}

type CosmosHeader struct {
	Header  types.Header
	Commit  *types.Commit
	Valsets []*types.Validator
}

var (
	SpanPrefixKey = []byte{0x36} // prefix key to store span
)

// GetSpanKey appends prefix to start block
func GetSpanKey(id uint64) []byte {
	return append(SpanPrefixKey, []byte(strconv.FormatUint(id, 10))...)
}

func main() {
	httpClient, _ := httpClient.New(TendermintRPCUrl, "/websocket")
	// err := httpClient.Start()
	// if err != nil {
	// 	panic(fmt.Sprintf("Error connecting to server %v", err))
	// }

	height := 100
	for i := 0; i < 50; i++ {
		height0 := int64(height + i)
		block0, err := helper.GetBlockWithClient(httpClient, height0)
		if err != nil {
			panic(err)
		}

		page := 0
		perPage := 50

		height1 := int64(height + i + 1)
		block1, err := helper.GetBlockWithClient(httpClient, height1)
		if err != nil {
			panic(err)
		}

		vals1, err := httpClient.Validators(context.Background(), &height1, &page, &perPage)
		if err != nil {
			panic(err)
		}
		commit1, err := httpClient.Commit(context.Background(), &height1)
		if err != nil {
			panic(err)
		}

		// resp, err := httpClient.ABCIQueryWithOptions(context.Background(), "/store/bor/key", GetSpanKey(1), client.ABCIQueryOptions{Prove: true})
		// if err != nil {
		// 	panic(err)
		// }

		// fmt.Println(resp)
		// return
		info := &CosmosEpochSwitchInfo{
			NextValidatorsHash: block0.NextValidatorsHash,
			ChainID:            block0.ChainID,
		}
		header := &CosmosHeader{
			Header:  block1.Header,
			Commit:  commit1.Commit,
			Valsets: vals1.Validators,
		}

		err = VerifyCosmosHeader(header, info)
		if err != nil {
			panic(fmt.Sprintf("VerifyCosmosHeader:%v", err))
		}
	}

}
