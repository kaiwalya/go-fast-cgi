package gofast

type fcgi_record_header struct {
	Version, RecordType uint8;
	RequestID, Length uint16;
	Padding, _ uint8;
}

type fcgi_record struct {
	fcgi_record_header ;
	Body [] byte;
}

type fcgi_begin_request_body struct {
	Role uint16;
	Flag uint8;
	_, _, _, _, _ uint8;
}

type fcgi_end_request_body struct{
	AppStatus uint32;
	ProtocolStatus uint8;
	_,_,_ uint8;
}

type fcgi_end_request_record struct {
	fcgi_record_header;
	fcgi_end_request_body;
}

type fcgi_begin_request_record struct {
	fcgi_record_header;
	fcgi_begin_request_body;
}

func (r * fcgi_end_request_record) Init() {
	r.Version = 1;
	r.RecordType = 3;
	r.Padding = 0;
	r.Length = 8;
}

func (r * fcgi_end_request_record) Set(requestID uint16, appStatus uint32, protocolStatus uint8) {
	r.RequestID = requestID;
	r.AppStatus = appStatus;
	r.ProtocolStatus = protocolStatus;
}
