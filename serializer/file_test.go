package serializer

import (
	"gobook/pb"
	"gobook/sample"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestFile(t *testing.T) {
	laptop1 := sample.NewLaptop()

	binaryFile := "../tmp/laptop.bin"
	jsonFile := "../tmp/laptop.json"

	err := WriteProtoToBinaryFile(binaryFile, laptop1)
	require.NoError(t, err)

	err = WriteProtobufToJsonFile(laptop1, jsonFile)
	require.NoError(t, err)

	laptop2 := &pb.Laptop{}

	err = ReadFromBinaryFile(binaryFile, laptop2)
	require.NoError(t, err)
	require.True(t, proto.Equal(laptop1, laptop2))
}
