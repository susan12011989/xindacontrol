package entity

type AnnouncementEntity struct {
    Type   string `json:"type"`
    Offset int    `json:"offset"`
    Length int    `json:"length"`
    Url    string `json:"url,omitempty"`
}

type AnnouncementReq struct {
    Text       string               `json:"text"`
    Entities   []AnnouncementEntity `json:"entities,omitempty"`
    Silent     bool                 `json:"silent"`
    NoForwards bool                 `json:"noforwards"`
}


