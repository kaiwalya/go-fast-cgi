package gofast

import (
	"net"
	"fmt"
	"bytes"
	"encoding/binary"
)

var request_handler_control_readComplete int;
var request_handler_control_writeComplete int;

func init() {
	request_handler_control_writeComplete = 1;
	request_handler_control_readComplete = 2;
}


type request_handler struct{
	Connection * net.TCPConn;
	RequestID uint16;
	mqRecords chan * fcgi_record;
	mqControl chan int;
	recordStack * record_stack;
	state int8;
	requestHeaders map[string] string;

	autoCloseConnection, writeComplete, readComplete bool;
}


func NewRequestHandler(conn * net.TCPConn, requestID uint16) * request_handler {
	var handler request_handler;

	handler.mqRecords = make(chan * fcgi_record, 10);
	handler.mqControl = make(chan int, 1);
	handler.state = 0;
	handler.recordStack = NewRecordStack();
	handler.requestHeaders = make(map[string]string);

	handler.autoCloseConnection = false;
	handler.writeComplete = false;
	handler.readComplete = false;

	go handler.processRequestRecords(conn, requestID);
	return &handler;
}


func (this * request_handler) temp_sendEmptyOutput() {
	var endOfStdout fcgi_record_header;
	endOfStdout.Version = 1;
	endOfStdout.RecordType = 6;
	endOfStdout.RequestID = this.RequestID;
	endOfStdout.Padding = 0;
	response := "Status: 201 Created\r\n\r\n\r\n";
	endOfStdout.Length = uint16(len([]byte(response)));
	//fmt.Println(endOfStdout);
	binary.Write(this.Connection, binary.BigEndian, endOfStdout);
	//fmt.Println(len([]byte(response)), ([]byte(response)));
	binary.Write(this.Connection, binary.LittleEndian, []byte(response));


	endOfStdout.Length = 0;
	//fmt.Println(endOfStdout);
	binary.Write(this.Connection, binary.BigEndian, endOfStdout);

	endOfStdout.RecordType = 7;
	binary.Write(this.Connection, binary.BigEndian, endOfStdout);

	var endOfRequest fcgi_end_request_record;
	endOfRequest.Init();
	endOfRequest.Set(this.RequestID, 0, 0);
	//fmt.Println(endOfRequest);
	binary.Write(this.Connection, binary.BigEndian, endOfRequest);

	this.mqControl <- request_handler_control_writeComplete;
}
func (this * request_handler) processRequestRecords_onRecord(conn * net.TCPConn, requestID uint16, record * fcgi_record) {
	if requestID != record.RequestID {
		panic("Wrond request to wrong processor");
	}

	state := this.state;
	recType := record.RecordType;
	recLen := record.Length;
	switch  {
	default:
		fmt.Println("processRequestRecords", "WIP, state", state, "recType", recType, "recLen", recLen);
		panic("processRequestRecords Unknown packet");
	//Begin
	case (state == 0 && recType == 1):
		buff := bytes.NewBuffer(record.Body);
		brb := new(fcgi_begin_request_body);
		binary.Read(buff, binary.BigEndian, brb)
		if (brb.Flag & 1) != 0 {
			this.autoCloseConnection = false;
		} else {
			this.autoCloseConnection = true;
		}
		//fmt.Println("processRequestRecords", "New Request", record, brb);
		this.Connection = conn;
		this.RequestID = requestID;
		state = 0;

	case (state == 0 && recType == 2):
		fmt.Println("processRequestRecords", "Abort Request");
		state = 0;

	//Accepting Params
	case (state == 0 && recType == 4):
		if recLen > 0 {
			//fmt.Println("processRequestRecords", "Headers Start");
			this.recordStack.push(record);
			this.recordStack.parseKeyValueStrings(this.requestHeaders);
			state = 4;
		} else {
			//fmt.Println("processRequestRecords", "Headers None");
			state = 0;
		}
	case (state == 4 && recType == 4):
		if recLen > 0 {
			//fmt.Println("processRequestRecords", "Headers Continue");
			this.recordStack.push(record);
			this.recordStack.parseKeyValueStrings(this.requestHeaders);
			state = 4;
		} else {
			//fmt.Println("processRequestRecords", "Headers End");
			this.recordStack.parseKeyValueStrings(this.requestHeaders);
			//for k,v := range this.requestHeaders {
			//	fmt.Println(k,v);
			//}
			state = 0;
		}

	//Accepting Stdin
	case (state == 0 && recType == 5):
		if recLen > 0 {
			//fmt.Println("processRequestRecords", "Stdin Start");
			fmt.Println(record.Body);
			state = 5;
		} else {
			//fmt.Println("processRequestRecords", "Stdin None");
			this.temp_sendEmptyOutput();
			state = 0;
		}
	case (state == 5 && recType == 5):
		if recLen > 0 {
			//fmt.Println("processRequestRecords", "Stdin Continue");
			state = 5;
		} else {
			//fmt.Println("processRequestRecords", "Stdin End");
			this.temp_sendEmptyOutput();	
			state = 0;
		}
	}
	this.state = state;
}
func (this * request_handler) processRequestRecords(conn * net.TCPConn, requestID uint16) {
	for {
		select {
			case record := <- this.mqRecords:
				//fmt.Println("Record", record.RequestID, record.RecordType);
				this.processRequestRecords_onRecord(conn, requestID, record);
				//fmt.Println("Record Processed", record.RequestID, record.RecordType);
			case control := <- this.mqControl:
				switch control{
				case request_handler_control_readComplete:
					this.readComplete = true;
				case request_handler_control_writeComplete:
					this.writeComplete = true;
				}
				if (this.autoCloseConnection && this.readComplete) {
					fmt.Println("Closing Read");
					this.Connection.CloseRead();
					//this.Connection.Close();
				}
				if (this.autoCloseConnection && this.writeComplete) {
					fmt.Println("Closing");
					this.Connection.Close();
					//this.Connection.Close();
				}

				if (this.writeComplete) {
					
					return;
				}
		}
	}
}

func (this * request_handler) NotifyReadComplete() {
	this.mqControl <- request_handler_control_readComplete;
}
