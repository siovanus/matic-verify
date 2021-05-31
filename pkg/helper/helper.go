package helper

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/pkg/errors"
	httpClient "github.com/tendermint/tendermint/rpc/client/http"
	tmTypes "github.com/tendermint/tendermint/types"
)

const (
	// CommitTimeout commit timeout
	CommitTimeout = 2 * time.Minute
)

// GetBlockWithClient get block through per height
func GetBlockWithClient(client *httpClient.HTTP, height int64) (*tmTypes.Block, error) {
	c, cancel := context.WithTimeout(context.Background(), CommitTimeout)
	defer cancel()

	// get block using client
	block, err := client.Block(c, &height)
	if err == nil && block != nil {
		return block.Block, nil
	}

	// subscriber
	subscriber := fmt.Sprintf("new-block-%v", height)

	// query for event
	query := tmTypes.QueryForEvent(tmTypes.EventNewBlock).String()

	// register for the next event of this type
	eventCh, err := client.Subscribe(c, subscriber, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to subscribe")
	}

	// unsubscribe query
	defer func() {
		if err := client.Unsubscribe(c, subscriber, query); err != nil {
			log.Fatal("GetBlockWithClient | Unsubscribe", "Error", err)
		}
	}()

	for {
		select {
		case event := <-eventCh:
			eventData := event.Data.(tmTypes.TMEventData)
			switch t := eventData.(type) {
			case tmTypes.EventDataNewBlock:
				if t.Block.Height == height {
					return t.Block, nil
				}
			default:
				return nil, errors.New("timed out waiting for event")
			}
		case <-c.Done():
			return nil, errors.New("timed out waiting for event")
		}
	}
}
