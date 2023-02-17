package serializer

import (
	"fmt"
	"os"

	"google.golang.org/protobuf/proto"
)

func WriteProtobufToJsonFile(message proto.Message, filename string) error {

	data, err := ProtobufToJson(message)
	if err != nil {
		return fmt.Errorf("cannot marshal proto message to json %w ", err)
	}

	err = os.WriteFile(filename, data, 0644)

	if err != nil {
		return fmt.Errorf("cannot write json to file %w ", err)
	}

	return nil
}

func WriteProtoToBinaryFile(filename string, message proto.Message) error {
	data, err := proto.Marshal(message)

	if err != nil {
		return fmt.Errorf("cannot marshal proto message to binary %w ", err)
	}

	err = os.WriteFile(filename, data, 0644)

	if err != nil {
		return fmt.Errorf("cannot write binary to file %w ", err)
	}

	return nil
}

func ReadFromBinaryFile(filename string, message proto.Message) error {
	data, err := os.ReadFile(filename)

	if err != nil {
		return fmt.Errorf("cannot read file %w ", err)
	}

	err = proto.Unmarshal(data, message)

	if err != nil {
		return fmt.Errorf("cannot unmarshal binary to proto message: %w", err)
	}

	return nil
}
