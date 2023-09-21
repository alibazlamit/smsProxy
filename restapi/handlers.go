package restapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"gitlab.com/devskiller-tasks/messaging-app-golang/smsproxy"
)

func sendSmsHandler(smsProxy smsproxy.SmsProxy) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		var sendSmsRequest smsproxy.SendMessage
		if err := json.NewDecoder(request.Body).Decode(&sendSmsRequest); err != nil {
			handleError(writer, http.StatusBadRequest, err)
			return
		}

		sendingResult, err := smsProxy.Send(sendSmsRequest)
		if err != nil {
			if validationErr, ok := err.(*smsproxy.ValidationError); ok {
				handleError(writer, http.StatusBadRequest, validationErr)
			} else {
				handleError(writer, http.StatusInternalServerError, err)
			}
			return
		}

		writer.WriteHeader(http.StatusAccepted)
		repoonse, err := json.Marshal(sendingResult)
		if err != nil {
			handleError(writer, http.StatusInternalServerError, err)
			return
		}
		writer.Write(repoonse)
	}
}

func getSmsStatusHandler(smsProxy smsproxy.SmsProxy) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		messageID, err := getMessageID(request.URL.RequestURI())
		if err != nil {
			handleError(writer, http.StatusInternalServerError, err)
			return
		}
		result, err := smsProxy.GetStatus(messageID.String())
		if err != nil {
			handleError(writer, http.StatusInternalServerError, err)
			return
		}

		responseBody, err := json.Marshal(SmsStatusResponse{Status: result})
		if err != nil {
			handleError(writer, http.StatusInternalServerError, err)
			return
		}

		if _, err = writer.Write(responseBody); err != nil {
			log.Println(errors.Wrapf(err, "cannot write http response").Error())
		}
	}
}

func getMessageID(uri string) (uuid.UUID, error) {
	uriParts := strings.Split(uri, "/")
	parse, err := uuid.Parse(uriParts[2])
	return parse, err
}

func handleError(writer http.ResponseWriter, status int, err error) {
	response := HttpErrorResponse{Error: err.Error()}
	jsonBody, err := json.Marshal(response)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		_, _ = writer.Write([]byte("Error serializing response"))
		log.Println(errors.Wrapf(err, "error serializing json response").Error())
	}
	writer.WriteHeader(status)
	_, err = writer.Write(jsonBody)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		log.Println(errors.Wrapf(err, "error writing HTTP response").Error())
	}
}
