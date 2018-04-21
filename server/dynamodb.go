package server

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/pkg/errors"
)

// saveNewInbox saves the passed in inbox to dynamodb
func (s *Server) saveNewInbox(i Inbox) error {
	av, err := dynamodbattribute.MarshalMap(i)

	if err != nil {
		return fmt.Errorf("putEmailToDB: failed to marshal struct to attribute value: %v", err)

	}

	_, err = s.dynDB.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String("emails"),
		Item:      av,
	})

	if err != nil {
		return fmt.Errorf("putEmailToDB: failed to put to dynamodb: %v", err)
	}

	return nil
}

// getInboxByID gets an email by id
func (s *Server) getInboxByID(id string) (Inbox, error) {
	var i Inbox

	o, err := s.dynDB.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
		},
		TableName: aws.String("emails"),
	})

	if err != nil {
		return Inbox{}, err
	}

	err = dynamodbattribute.UnmarshalMap(o.Item, &i)

	if err != nil {
		return Inbox{}, err
	}

	return i, nil
}

// emailExists checks to see if the given email address already exists in our db. It will only return
// false if we can explicitly verify the email doesn't exist.
func (s *Server) emailExists(a string) (bool, error) {
	q := &dynamodb.QueryInput{
		KeyConditionExpression: aws.String("email_address = :e"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":e": {
				S: aws.String(a),
			},
		},
		IndexName: aws.String("email_address-index"),
		TableName: aws.String("emails"),
	}

	res, err := s.dynDB.Query(q)

	if err != nil {
		return false, err
	}

	if len(res.Items) == 0 {
		return false, nil
	}

	return true, nil
}

// setInboxCreated updates dynamodb and sets the email as created and adds a mailgun route
func (s *Server) setInboxCreated(i Inbox) error {
	u := &dynamodb.UpdateItemInput{
		ExpressionAttributeNames: map[string]*string{
			"#F": aws.String("failed_to_create"),
			"#M": aws.String("mg_routeid"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":f": {
				BOOL: aws.Bool(false),
			},
			":m": {
				S: aws.String(i.MGRouteID),
			},
		},
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(i.ID),
			},
		},
		TableName:        aws.String("emails"),
		UpdateExpression: aws.String("SET #F = :f, #M = :m"),
	}

	_, err := s.dynDB.UpdateItem(u)

	if err != nil {
		return fmt.Errorf("setInboxCreated: failed to mark email as created: %v", err)
	}

	return err
}

// saveMessage saves a given message to dynamodb
func (s *Server) saveNewMessage(m Message) error {
	mv, err := dynamodbattribute.MarshalMap(m)

	if err != nil {
		return fmt.Errorf("saveMessage: failed to marshal struct to attribute value: %v", err)
	}

	_, err = s.dynDB.PutItem(&dynamodb.PutItemInput{
		TableName: aws.String("messages"),
		Item:      mv,
	})

	if err != nil {
		return fmt.Errorf("saveMessage: failed to put to dynamodb: %v", err)
	}

	return nil
}

//getAllMessagesByInboxID gets all messages in the given inbox
func (s *Server) getAllMessagesByInboxID(i string) ([]Message, error) {
	var m []Message

	qi := &dynamodb.QueryInput{
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":id": {
				S: aws.String(i),
			},
		},
		KeyConditionExpression: aws.String("inbox_id = :id"),
		TableName:              aws.String("messages"),
	}

	res, err := s.dynDB.Query(qi)

	if err != nil {
		return []Message{}, fmt.Errorf("getAllMessagesByInboxID: failed to query dynamodb: %v", err)
	}

	err = dynamodbattribute.UnmarshalListOfMaps(res.Items, &m)

	if err != nil {
		return []Message{}, fmt.Errorf("getAllMessagesByInboxID: failed to unmarshal result: %v", err)
	}

	return m, nil
}

var errMessageDoesntExist = errors.New("Error: message doesn't exist")

//getSingularMessage gets a single message by the given inbox and message id
func (s *Server) getMessageByID(i, m string) (Message, error) {
	var msg Message

	o, err := s.dynDB.GetItem(&dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"inbox_id": {
				S: aws.String(i),
			},
			"message_id": {
				S: aws.String(m),
			},
		},
		TableName: aws.String("messages"),
	})

	if err != nil {
		return Message{}, fmt.Errorf("getMessageByID: failed to get message: %v", err)
	}

	err = dynamodbattribute.UnmarshalMap(o.Item, &msg)

	if err != nil {
		return Message{}, fmt.Errorf("getMessageByID: failed to unmarshal message: %v", err)
	}

	if strings.Compare(msg.ID, "") == 0 {
		return Message{}, errMessageDoesntExist
	}

	return msg, nil
}
