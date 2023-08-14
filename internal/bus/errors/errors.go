// Package errors provides string codes for error instantiation.

package errors

const (
	AMQPConnectionError          = "could not connect to AMQP"
	AMQPChannelOpeningError      = "could not open an AMQP channel"
	AMQPSettingQosError          = "could not set QoS"
	AMQPExchangeDeclarationError = "could not declare an exchange"
	AMQPQueueDeclarationError    = "could not declare a queue"
	AMQPInitiationError          = "could not initialize AMQP"
	AMQPSerialisationError       = "could not serialize a message"
	AMQPPublishingError          = "could not publish a message"
	AMQPConsumingError           = "failed to start consuming messages from queue"
	AMQPAckError                 = "failed to acknowledge message"
	AMQPMessageProcessingError   = "failed to process message"
	AMQPSendingError             = "failed to send message"
	AMQPListeningError           = "failed to listen to queue"
	AMQPUnmarshallingError       = "failed to unmarshall message"
	AMQPMarshallingError         = "failed to marshall message"
	AMQPHandlerValidationError   = "failed to run validation for AMQP-derived query"
	AMQPHandlerProcessingError   = "failed to run processing for AMQP-derived query"
)
