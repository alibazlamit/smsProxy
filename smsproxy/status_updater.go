package smsproxy

import (
	"fmt"

	"gitlab.com/devskiller-tasks/messaging-app-golang/fastsmsing"
)

type statusUpdateError struct {
	err       error
	messageID fastsmsing.MessageID
	status    fastsmsing.MessageStatus
}

type statusUpdater struct {
	C          chan map[fastsmsing.MessageID]fastsmsing.MessageStatus
	Errors     chan statusUpdateError
	repository repository
}

func newStatusUpdater(repository repository) statusUpdater {
	return statusUpdater{
		C:          make(chan map[string]fastsmsing.MessageStatus),
		repository: repository,
		Errors:     make(chan statusUpdateError)}
}

func (u statusUpdater) Start() {
	go func() {
		for statusMap := range u.C {
			for messageID, status := range statusMap {
				internalStatus, err := mapToInternalStatus(status)
				if err != nil {
					u.Errors <- statusUpdateError{
						err:       err,
						messageID: messageID,
						status:    status,
					}
					continue
				}

				if err := u.repository.update(MessageID(messageID), internalStatus); err != nil {
					u.Errors <- statusUpdateError{
						err:       err,
						messageID: messageID,
						status:    status,
					}
				}
			}
		}
	}()
}

func mapToInternalStatus(status fastsmsing.MessageStatus) (MessageStatus, error) {
	for _, mappableStatus := range allStatuses {
		if string(status) == string(mappableStatus) {
			return mappableStatus, nil
		}
	}
	return "", fmt.Errorf("cannot map status %s to any known status", status)
}
