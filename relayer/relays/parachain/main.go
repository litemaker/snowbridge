package parachain

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/snowfork/snowbridge/relayer/chain/ethereum"
	"github.com/snowfork/snowbridge/relayer/chain/parachain"
	"github.com/snowfork/snowbridge/relayer/chain/relaychain"
	"github.com/snowfork/snowbridge/relayer/crypto/secp256k1"

	log "github.com/sirupsen/logrus"
)

type Relay struct {
	config                *Config
	parachainConn         *parachain.Connection
	relaychainConn        *relaychain.Connection
	ethereumConn          *ethereum.Connection
	ethereumChannelWriter *EthereumChannelWriter
	beefyListener         *BeefyListener
}

func NewRelay(config *Config, keypair *secp256k1.Keypair) (*Relay, error) {

	log.Info("Creating worker")

	parachainConn := parachain.NewConnection(config.Parachain.Endpoint, nil)
	relaychainConn := relaychain.NewConnection(config.Polkadot.Endpoint)
	ethereumConn := ethereum.NewConnection(config.Ethereum.Endpoint, keypair)

	// channel for messages from beefy listener to ethereum writer
	var messagePackages = make(chan MessagePackage, 1)

	ethereumChannelWriter, err := NewEthereumChannelWriter(
		config,
		ethereumConn,
		messagePackages,
	)
	if err != nil {
		return nil, err
	}

	beefyListener := NewBeefyListener(
		config,
		ethereumConn,
		relaychainConn,
		parachainConn,
		messagePackages,
	)

	return &Relay{
		config:                config,
		parachainConn:         parachainConn,
		relaychainConn:        relaychainConn,
		ethereumConn:          ethereumConn,
		ethereumChannelWriter: ethereumChannelWriter,
		beefyListener:         beefyListener,
	}, nil
}

func (relay *Relay) Start(ctx context.Context, eg *errgroup.Group) error {
	log.Info("Starting worker")

	if relay.beefyListener == nil || relay.ethereumChannelWriter == nil {
		return fmt.Errorf("Sender and/or receiver need to be set before starting chain")
	}

	err := relay.parachainConn.Connect(ctx)
	if err != nil {
		return err
	}

	err = relay.ethereumConn.Connect(ctx)
	if err != nil {
		return err
	}

	err = relay.relaychainConn.Connect(ctx)
	if err != nil {
		return err
	}

	eg.Go(func() error {
		if relay.ethereumChannelWriter != nil {
			log.Info("Starting Writer")
			err = relay.ethereumChannelWriter.Start(ctx, eg)
			if err != nil {
				return err
			}
		}
		return nil
	})

	eg.Go(func() error {
		if relay.beefyListener != nil {
			log.Info("Starting Beefy Listener")
			err = relay.beefyListener.Start(ctx, eg)
			if err != nil {
				return err
			}
		}
		return nil
	})

	return nil
}

func (relay *Relay) Stop() {
	if relay.parachainConn != nil {
		relay.parachainConn.Close()
	}
	if relay.relaychainConn != nil {
		relay.relaychainConn.Close()
	}
	if relay.ethereumConn != nil {
		relay.ethereumConn.Close()
	}
}