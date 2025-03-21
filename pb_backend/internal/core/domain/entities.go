package domain

type Color [3]uint

type Pixel struct {
	X       uint   `json:"x"`
	Y       uint   `json:"y"`
	Color   []uint `json:"color"`
	Userid  int    `json:"userid"`
	Faculty string `json:"faculty"`
}

type RedisPixel struct {
	UserId    int    `json:"userid"`
	Faculty   string `json:"faculty"`
	Color     []uint `json:"color"`
	Timestamp int64  `json:"timestamp"`
}

type HeatMapUnit struct {
	X   uint
	Y   uint
	Len uint
}

type UserStats struct {
	TotalPixelsPlaced int `bson:"total_pixels_placed"`
	ActivePixels      int `bson:"active_pixels"`
}

type User struct {
	ID          int       `json:"id" bson:"_id"`
	FirstName   string    `json:"name" bson:"first_name"`
	LastName    string    `json:"surname" bson:"last_name"`
	AccessToken string    `json:"token" bson:"access_token"`
	Faculty     string    `json:"faculty" bson:"faculty"`
	Stats       UserStats `bson:"stats"`
}

type Image struct {
	Height uint
	Width  uint
	Data   []Pixel
}

type BroadcastStats struct {
	ID                int    `json:"id" bson:"_id"`
	FirstName         string `json:"name" bson:"first_name"`
	LastName          string `json:"surname" bson:"last_name"`
	TotalPixelsPlaced int    `json:"total_pixels_placed" bson:"total_pixels_placed"`
	ActivePixels      int    `json:"active_pixels" bson:"active_pixels"`
}
