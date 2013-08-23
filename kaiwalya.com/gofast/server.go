package gofast;

import (
	"net"
	"fmt"
	"encoding/binary"
//	"runtime"
	"io"
)

type IServer interface {
	ManageNewConnection(conn * net.TCPConn);
}

type server struct {
	connectionCount int;
}

func (this * server) procConnectionHandler(conn * net.TCPConn) {
	//fmt.Println("procConnectionHandler:", "New Connection", conn, "at", conn.LocalAddr(), "from", conn.RemoteAddr());
	var err error;
	mapRequest := map[uint16] * request_handler {};
	for {
		
		fcgiRecord := new (fcgi_record);
		err = binary.Read(conn, binary.BigEndian, &fcgiRecord.fcgi_record_header);
		if err != nil {
			if (err.Error() != "EOF") {
				fmt.Println("procConnectionHandler", "conn", conn, "Error Reading Data, Quiting", err);
			} else{
				if len(mapRequest) != 1 {
					fmt.Println("procConnectionHandler", "EOF on wrong kind of connection");
					panic("procConnectionHandler: EOF on wrong kind of connection");
				}
				for _,v := range mapRequest {
					v.NotifyReadComplete();
				}
			}
			return;
		}
		
		fcgiRecord.Body = make([]byte, fcgiRecord.Length, fcgiRecord.Length);
		_, err = io.ReadFull(conn, fcgiRecord.Body)
		if err != nil {
			fmt.Println("procConnectionHandler", "conn", conn, "Error Reading Data, Quiting", err);
			return;
		}

		padding:= fcgiRecord.Padding
		//Send the record to the handler
		{
			rid := fcgiRecord.RequestID;
			handler := mapRequest[rid];

			if fcgiRecord.RecordType == uint8(1) {
				handler = NewRequestHandler(conn, rid);
				mapRequest[rid] = handler;
				//fmt.Println("processor_record_dispatch:", id, "New Handler", handler);
			}
			handler.mqRecords <- fcgiRecord;
		}

		if padding > 0 {
			discard := make([]byte, padding, padding);
			io.ReadFull(conn, discard);
			if (err != nil) {
				fmt.Println("procConnectionHandler", "conn", conn, "Error Reading Data, Quiting", err);
				return;
			}
		}
	}
}

func (this * server) ManageNewConnection(conn * net.TCPConn) {
	this.connectionCount++;
	fmt.Println("Connection Count", this.connectionCount);
	go this.procConnectionHandler(conn);
}

func NewServer() (IServer, error) {
	var s server;
	s.connectionCount = 0;
	return &s, nil;
}
