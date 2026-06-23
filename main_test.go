package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func Test_getEcsMetadata(t *testing.T) {
	const want = "127.0.0.1"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"Name":"curl","Networks":[{"IPv4Addresses":["` + want + `"]}]}`))
	})
	server := httptest.NewServer(handler)
	defer server.Close()

	t.Setenv("ECS_CONTAINER_METADATA_URI_V4", server.URL)
	t.Setenv("ECS_CONTAINER_METADATA_URI", "")

	got, err := getEcsMetadata()
	if err != nil {
		t.Errorf("getEcsMetadata() error = %v", err)
		return
	}
	if got.Networks[0].IPv4Addresses[0] != want {
		t.Errorf("getEcsMetadata() = %v, want %v", got, want)
	}
}

type mockEC2Client struct {
	output *ec2.DescribeNetworkInterfacesOutput
	err    error
	input  *ec2.DescribeNetworkInterfacesInput
}

func (m *mockEC2Client) DescribeNetworkInterfaces(_ context.Context, params *ec2.DescribeNetworkInterfacesInput, _ ...func(*ec2.Options)) (*ec2.DescribeNetworkInterfacesOutput, error) {
	m.input = params
	return m.output, m.err
}

func Test_getEcsPublicIP(t *testing.T) {
	t.Parallel()

	privateIPs := []string{"10.0.0.11", "10.0.0.12"}
	client := &mockEC2Client{
		output: &ec2.DescribeNetworkInterfacesOutput{
			NetworkInterfaces: []ec2types.NetworkInterface{
				{
					PrivateIpAddresses: []ec2types.NetworkInterfacePrivateIpAddress{
						{
							PrivateIpAddress: stringPtr("10.0.0.12"),
							Association: &ec2types.NetworkInterfaceAssociation{
								PublicIp: stringPtr("54.1.2.3"),
							},
						},
					},
				},
			},
		},
	}

	got, err := getEcsPublicIP(context.Background(), client, privateIPs)
	if err != nil {
		t.Fatalf("getEcsPublicIP() error = %v", err)
	}
	if got != "54.1.2.3" {
		t.Fatalf("getEcsPublicIP() = %v, want %v", got, "54.1.2.3")
	}

	if client.input == nil {
		t.Fatal("DescribeNetworkInterfaces was not called")
	}
	if len(client.input.Filters) != 1 {
		t.Fatalf("DescribeNetworkInterfaces filters = %d, want 1", len(client.input.Filters))
	}
	if *client.input.Filters[0].Name != "addresses.private-ip-address" {
		t.Fatalf("DescribeNetworkInterfaces filter name = %q", *client.input.Filters[0].Name)
	}
	if !reflect.DeepEqual(client.input.Filters[0].Values, privateIPs) {
		t.Fatalf("DescribeNetworkInterfaces filter values = %v, want %v", client.input.Filters[0].Values, privateIPs)
	}
}

func Test_getEcsPublicIP_NoPrivateIPs(t *testing.T) {
	t.Parallel()

	_, err := getEcsPublicIP(context.Background(), &mockEC2Client{}, nil)
	if err == nil {
		t.Fatal("getEcsPublicIP() error = nil, want error")
	}
}

func Test_getEcsPublicIP_NoPublicIPFound(t *testing.T) {
	t.Parallel()

	client := &mockEC2Client{
		output: &ec2.DescribeNetworkInterfacesOutput{
			NetworkInterfaces: []ec2types.NetworkInterface{
				{
					PrivateIpAddresses: []ec2types.NetworkInterfacePrivateIpAddress{
						{
							PrivateIpAddress: stringPtr("10.0.0.11"),
						},
					},
				},
			},
		},
	}

	_, err := getEcsPublicIP(context.Background(), client, []string{"10.0.0.11"})
	if err == nil {
		t.Fatal("getEcsPublicIP() error = nil, want error")
	}
}

func stringPtr(v string) *string {
	return &v
}
