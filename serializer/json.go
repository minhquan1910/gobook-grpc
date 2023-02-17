package serializer

import (
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func ProtobufToJson(message proto.Message) ([]byte, error) {
	marshaller := protojson.MarshalOptions{
		UseProtoNames: false,
		Indent:        "	",
	}
	marshaler, error := marshaller.Marshal(message)

	return marshaler, error
}

func JsonToProtobufMessage(data string, message proto.Message) error {
	return protojson.Unmarshal([]byte(data), message)
}
