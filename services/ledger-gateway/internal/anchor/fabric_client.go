package anchor

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hyperledger/fabric-gateway/pkg/client"
	"github.com/hyperledger/fabric-gateway/pkg/identity"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// FabricConfig locates the Fabric network and the gateway's enrollment
// material. Certificates and keys are mounted at deploy time — never
// committed to the repository.
type FabricConfig struct {
	PeerEndpoint string
	MSPID        string
	CertPath     string
	KeyPath      string // file, or an MSP keystore directory
	TLSCertPath  string // empty → plaintext gRPC (local dev network only)
	Channel      string
	Chaincode    string
}

// gatewayContract adapts the Fabric gateway client to fabricSubmitter,
// surfacing the transaction id the backend records as its reference.
type gatewayContract struct {
	contract *client.Contract
}

func (g gatewayContract) Submit(name string, args ...string) ([]byte, string, error) {
	proposal, err := g.contract.NewProposal(name, client.WithArguments(args...))
	if err != nil {
		return nil, "", err
	}
	transaction, err := proposal.Endorse()
	if err != nil {
		return nil, "", err
	}
	commit, err := transaction.Submit()
	if err != nil {
		return nil, "", err
	}
	status, err := commit.Status()
	if err != nil {
		return nil, "", err
	}
	if !status.Successful {
		return nil, "", fmt.Errorf("transaction %s failed with status %d", status.TransactionID, int32(status.Code))
	}
	return transaction.Result(), status.TransactionID, nil
}

// ConnectFabric dials the peer, builds a gateway identity from the enrollment
// material, and returns a FabricBackend plus a close function.
func ConnectFabric(config FabricConfig) (*FabricBackend, func(), error) {
	certPEM, err := os.ReadFile(config.CertPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading enrollment certificate: %w", err)
	}
	certificate, err := identity.CertificateFromPEM(certPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing enrollment certificate: %w", err)
	}
	id, err := identity.NewX509Identity(config.MSPID, certificate)
	if err != nil {
		return nil, nil, err
	}

	keyPEM, err := os.ReadFile(resolveKeyPath(config.KeyPath))
	if err != nil {
		return nil, nil, fmt.Errorf("reading private key: %w", err)
	}
	privateKey, err := identity.PrivateKeyFromPEM(keyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing private key: %w", err)
	}
	sign, err := identity.NewPrivateKeySign(privateKey)
	if err != nil {
		return nil, nil, err
	}

	transport := insecure.NewCredentials()
	if config.TLSCertPath != "" {
		tlsCredentials, err := credentials.NewClientTLSFromFile(config.TLSCertPath, "")
		if err != nil {
			return nil, nil, fmt.Errorf("loading peer TLS certificate: %w", err)
		}
		transport = tlsCredentials
	}
	connection, err := grpc.NewClient(config.PeerEndpoint, grpc.WithTransportCredentials(transport))
	if err != nil {
		return nil, nil, fmt.Errorf("dialing peer: %w", err)
	}

	gateway, err := client.Connect(id, client.WithSign(sign), client.WithClientConnection(connection))
	if err != nil {
		_ = connection.Close()
		return nil, nil, fmt.Errorf("connecting Fabric gateway: %w", err)
	}

	contract := gateway.GetNetwork(config.Channel).GetContract(config.Chaincode)
	closer := func() {
		_ = gateway.Close()
		_ = connection.Close()
	}
	return NewFabricBackend(gatewayContract{contract: contract}, config.Chaincode, config.Channel), closer, nil
}

// resolveKeyPath accepts either a key file or an MSP keystore directory
// (cryptogen names keys non-deterministically inside keystore/).
func resolveKeyPath(path string) string {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return path
	}
	entries, err := os.ReadDir(path)
	if err != nil || len(entries) == 0 {
		return path
	}
	return filepath.Join(path, entries[0].Name())
}
