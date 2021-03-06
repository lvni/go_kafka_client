/* Licensed to the Apache Software Foundation (ASF) under one or more
 contributor license agreements.  See the NOTICE file distributed with
 this work for additional information regarding copyright ownership.
 The ASF licenses this file to You under the Apache License, Version 2.0
 (the "License"); you may not use this file except in compliance with
 the License.  You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License. */

package go_kafka_client

import (
	"github.com/Shopify/sarama"
	"testing"
	"time"
)

func TestMessageBuffer(t *testing.T) {
	askNextTimeout := 2 * time.Second

	config := DefaultConsumerConfig()
	config.FetchBatchSize = 5
	config.FetchBatchTimeout = 3 * time.Second

	out := make(chan []*Message)
	topicPartition := TopicAndPartition{"fakeTopic", 0}
	askNextBatch := make(chan TopicAndPartition)
	disconnectChannelsForPartition := make(chan TopicAndPartition)
	buffer := newMessageBuffer(topicPartition, out, config, askNextBatch, disconnectChannelsForPartition)

	receiveNoMessages(t, 4*time.Second, out)

	go buffer.addBatch(generateBatch(topicPartition, 1))
	expectAskNext(t, askNextBatch, askNextTimeout)
	receiveN(t, 1, 4*time.Second, out)

	go buffer.addBatch(generateBatch(topicPartition, config.FetchBatchSize))
	receiveN(t, config.FetchBatchSize, 4*time.Second, out)
	expectAskNext(t, askNextBatch, askNextTimeout)

	go buffer.addBatch(generateBatch(topicPartition, 1))
	expectAskNext(t, askNextBatch, askNextTimeout)

	go func() {
		select {
		case <-disconnectChannelsForPartition:
		case <-time.After(2 * time.Second):
			t.Error("Failed to receive 'ask next'")
		}
	}()
	buffer.stop()
	receiveNoMessages(t, 4*time.Second, out)
}

func expectAskNext(t *testing.T, askNext chan TopicAndPartition, timeout time.Duration) {
	select {
	case <-askNext:
		Trace("test", "Got asknext")
	case <-time.After(timeout):
		t.Error("Failed to receive 'ask next'")
	}
}

func generateBatch(topicPartition TopicAndPartition, size int) *TopicPartitionData {
	messages := make([]*sarama.MessageBlock, 0)
	for i := 0; i < size; i++ {
		messages = append(messages, &sarama.MessageBlock{int64(i), &sarama.Message{}})
	}

	return &TopicPartitionData{
		TopicPartition: topicPartition,
		Data: &sarama.FetchResponseBlock{
			MsgSet: sarama.MessageSet{
				Messages: messages,
			},
		},
	}
}
