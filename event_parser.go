package main

import (
	"fmt"
	"reflect"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

type eventParser struct {
	eventSignatureHash common.Hash
	eventABI           abi.Event
}

func NewEventParse(abi abi.ABI, eventName string) (eventParser, error) {
	var instance eventParser

	eventABI, exist := abi.Events[eventName]
	if !exist {
		return instance, fmt.Errorf("event '%s' not found in ABI", eventName)
	}
	eventSignature := []byte(eventABI.Sig)
	eventSignatureHash := crypto.Keccak256Hash(eventSignature)

	return eventParser{eventSignatureHash, eventABI}, nil
}

func (parser eventParser) CanParse(a types.Log) bool {
	return len(a.Topics) > 0 && a.Topics[0] == parser.eventSignatureHash
}

func (parser eventParser) Parse(a types.Log, target interface{}) error {
	if parser.CanParse(a) {
		data, err := parser.eventABI.Inputs.Unpack(a.Data)
		if err != nil {
			return err
		}
		return populateStruct(data, target)
	}
	return fmt.Errorf("unsupported log")
}

func populateStruct(data []interface{}, target interface{}) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Ptr || targetValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("target must be a pointer to a struct")
	}

	targetValue = targetValue.Elem()
	targetType := targetValue.Type()

	if len(data) != targetType.NumField() {
		return fmt.Errorf("mismatch between data length (%d) and number of struct fields (%d)", len(data), targetType.NumField())
	}

	for i := 0; i < targetType.NumField(); i++ {
		field := targetValue.Field(i)
		if !field.CanSet() {
			return fmt.Errorf("cannot set field %s", targetType.Field(i).Name)
		}

		dataValue := reflect.ValueOf(data[i])
		if !dataValue.Type().ConvertibleTo(field.Type()) {
			return fmt.Errorf("cannot convert %v to %v for field %s", dataValue.Type(), field.Type(), targetType.Field(i).Name)
		}

		field.Set(dataValue.Convert(field.Type()))
	}

	return nil
}
