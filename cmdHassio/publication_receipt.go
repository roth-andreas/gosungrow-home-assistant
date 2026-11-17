package cmdHassio

import "time"

// A receipt proves only the local MQTT operation completed. It cannot prove
// Home Assistant created an entity or that a dashboard rendered its value.
type PublicationStage struct {
	Topic       string
	Status      string
	CompletedAt time.Time
}

type SensorPublicationReceipt struct {
	UniqueID string
	Config   PublicationStage
	State    PublicationStage
}

func (m *Mqtt) SensorPublicationReceipt(config EntityConfig) SensorPublicationReceipt {
	id := JoinStringsForId(m.DeviceName, config.FullId)
	return SensorPublicationReceipt{
		UniqueID: id,
		Config:   PublicationStage{Topic: JoinStringsForTopic(m.Prefix, LabelSensor, m.ClientId, id, "config"), Status: "not_attempted"},
		State:    PublicationStage{Topic: JoinStringsForTopic(m.Prefix, LabelSensor, m.ClientId, id, "state"), Status: "not_attempted"},
	}
}
