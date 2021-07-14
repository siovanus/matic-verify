package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strconv"

	"github.com/tendermint/tendermint/libs/common"
	"github.com/tendermint/tendermint/rpc/client"
	"github.com/tendermint/tendermint/types"
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
	if myHeader.Commit.Height() != myHeader.Header.Height {
		return fmt.Errorf("VerifyCosmosHeader, commit height is not right! commit height: %d, "+
			"header height: %d", myHeader.Commit.Height(), myHeader.Header.Height)
	}
	if !bytes.Equal(myHeader.Commit.BlockID.Hash, myHeader.Header.Hash()) {
		return fmt.Errorf("VerifyCosmosHeader, commit hash is not right!, commit block hash: %s,"+
			" header hash: %s", myHeader.Commit.BlockID.Hash.String(), hex.EncodeToString(valset.Hash()))
	}
	if err := myHeader.Commit.ValidateBasic(); err != nil {
		return fmt.Errorf("VerifyCosmosHeader, commit is not right! err: %s", err.Error())
	}
	if valset.Size() != myHeader.Commit.Size() {
		return fmt.Errorf("VerifyCosmosHeader, the size of precommits is not right!")
	}
	talliedVotingPower := int64(0)
	for _, commitSig := range myHeader.Commit.Precommits {
		idx := commitSig.ValidatorIndex
		_, val := valset.GetByIndex(idx)
		if val == nil {
			return fmt.Errorf("VerifyCosmosHeader, validator %d doesn't exist!", idx)
		}
		if commitSig.Type != types.PrecommitType {
			return fmt.Errorf("VerifyCosmosHeader, commitSig.Type(%d) wrong", commitSig.Type)
		}
		// Validate signature.
		precommitSignBytes := myHeader.Commit.VoteSignBytes(info.ChainID, idx)
		if !val.PubKey.VerifyBytes(precommitSignBytes, commitSig.Signature) {
			return fmt.Errorf("VerifyCosmosHeader, Invalid commit -- invalid signature: %v", commitSig)
		}
		// Good precommit!
		if myHeader.Commit.BlockID.Equals(myHeader.Commit.BlockID) {
			talliedVotingPower += val.VotingPower
		}
	}
	if talliedVotingPower <= valset.TotalVotingPower()*2/3 {
		return fmt.Errorf("VerifyCosmosHeader, voteing power is not enough!")
	}

	return nil
}

type CosmosEpochSwitchInfo struct {
	NextValidatorsHash common.HexBytes
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
	client := client.NewHTTP(TendermintRPCUrl, "/websocket")
	// err := httpClient.Start()
	// if err != nil {
	// 	panic(fmt.Sprintf("Error connecting to server %v", err))
	// }

	height := 100
	for i := 0; i < 50; i++ {
		height0 := int64(height + i)
		commit0, err := client.Commit(&height0)
		if err != nil {
			panic(err)
		}

		height1 := int64(height + i + 1)

		vals1, err := client.Validators(&height1)
		if err != nil {
			panic(err)
		}

		commit1, err := client.Commit(&height1)
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
			NextValidatorsHash: commit0.NextValidatorsHash,
			ChainID:            commit0.ChainID,
		}

		header := &CosmosHeader{
			Header:  *commit1.Header,
			Commit:  commit1.Commit,
			Valsets: vals1.Validators,
		}

		err = VerifyCosmosHeader(header, info)
		if err != nil {
			panic(fmt.Sprintf("VerifyCosmosHeader:%v", err))
		} else {
			fmt.Println("VerifyCosmosHeader ok")
		}
	}

}
