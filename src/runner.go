package beerunner

// #include <TargetConditionals.h>
import (
	"C"
	
	"context"
	
	"os"
	"crypto/ecdsa"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethersphere/bee/pkg/crypto"
	"github.com/ethersphere/bee/pkg/keystore"
	memkeystore "github.com/ethersphere/bee/pkg/keystore/mem"
	"github.com/ethersphere/bee/pkg/node"
	"github.com/ethersphere/bee/pkg/logging"
	"github.com/ethersphere/bee/pkg/resolver/multiresolver"
	"github.com/ethersphere/bee/pkg/swarm"
	
	"github.com/sirupsen/logrus"
)

type program struct {
	start func()
	stop  func()
}

type signerConfig struct {
	signer           crypto.Signer
	address          swarm.Address
	publicKey        *ecdsa.PublicKey
	libp2pPrivateKey *ecdsa.PrivateKey
	pssPrivateKey    *ecdsa.PrivateKey
}

func StartBee() (err error) {
	var keystore keystore.Service
	keystore = memkeystore.New()
	var signer crypto.Signer
	var address swarm.Address
	var publicKey *ecdsa.PublicKey
	var logger logging.Logger
	logger = logging.New(os.Stdout, logrus.DebugLevel)
	
	password := "password"
	
	swarmPrivateKey, _, err := keystore.Key("swarm", password)
	if err != nil {
		return fmt.Errorf("swarm key: %w", err)
	}
	signer = crypto.NewDefaultSigner(swarmPrivateKey)
	publicKey = &swarmPrivateKey.PublicKey
	address, err = crypto.NewOverlayAddress(*publicKey, 1)
	if err != nil {
		return err
	}
	
	libp2pPrivateKey, _, err := keystore.Key("libp2p", password)
	pssPrivateKey, _, err := keystore.Key("pss", password)
	
	// overlayEthAddress, err := signer.EthereumAddress()
	
	var resolverCfgs []multiresolver.ConnectionConfig
	
	logger.Infof("Starting Bee Node")
	
	b, err := node.NewBee(
		":1634", 
		address, 
		*publicKey, 
		signer, 
		1, 
		logger, 
		libp2pPrivateKey, 
		pssPrivateKey, 
		node.Options{
			DataDir:                    ".bee",
			CacheCapacity:              1000000,
			DBOpenFilesLimit:           200,
			DBBlockCacheCapacity:       32*1024*1024,
			DBWriteBufferSize:          32*1024*1024,
			DBDisableSeeksCompaction:   false,
			APIAddr:                    ":1633",
			DebugAPIAddr:               ":1635",
			Addr:                       ":1634",
			NATAddr:                    "",
			EnableWS:                   false,
			EnableQUIC:                 false,
			WelcomeMessage:             "",
			Bootnodes:                  []string{"/dnsaddr/bootnode.ethswarm.org"},
			CORSAllowedOrigins:         []string{},
			Standalone:                 false,
			TracingEnabled:             false,
			TracingEndpoint:            "",
			TracingServiceName:         "",
			Logger:                     logger,
			GlobalPinningEnabled:       false,
			PaymentThreshold:           "10000000000000",
			PaymentTolerance:           "10000000000000",
			PaymentEarly:               "10000000000000",
			ResolverConnectionCfgs:     resolverCfgs,
			GatewayMode:                false,
			BootnodeMode:               false,
			SwapEndpoint:               "ws://localhost:8546",
			SwapFactoryAddress:         "",
			SwapLegacyFactoryAddresses: nil,
			SwapInitialDeposit:         "10000000000000000",
			SwapEnable:                 true,
			FullNodeMode:               false,
			Transaction:                "",
			PostageContractAddress:     "",
			BlockTime:                  15,
			DeployGasPrice:             "",
		},
	)
	
	if b != nil {
		logger.Infof("Yup")
	}
	
	if err != nil {
		return err
	}
	
	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, syscall.SIGINT, syscall.SIGTERM)
	
	p := &program{
		start: func() {
			// Block main goroutine until it is interrupted
			sig := <-interruptChannel

			logger.Debugf("received signal: %v", sig)
			logger.Info("shutting down")
		},
		stop: func() {
			// Shutdown
			done := make(chan struct{})
			go func() {
				defer close(done)

				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()

				if err := b.Shutdown(ctx); err != nil {
					logger.Errorf("shutdown: %v", err)
				}
			}()

			// If shutdown function is blocking too long,
			// allow process termination by receiving another signal.
			select {
			case sig := <-interruptChannel:
				logger.Debugf("received signal: %v", sig)
			case <-done:
			}
		},
	}
	
	p.start()
	p.stop()
	
	return nil
}

func main() {
	StartBee()
}