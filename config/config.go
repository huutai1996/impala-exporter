package config

type Memory struct {
	Name            string   `json:"name"`
	ModelerType     string   `json:"modelerType"`
	LastGcInfo      GCInfo   `json:"LastGcInfo"`
	CollectionCount float64  `json:"collectionCount"`
	CollectionTime  float64  `json:"collectionTime"`
	ShortName       string   `json:"Name"`
	Valid           bool     `json:"Valid"`
	MemoryPoolName  []string `json:"MemoryPoolName"`
	ObjectName      string   `json:"ObjectName"`
}

type GCInfo struct {
	GcThreadCount       float64          `json:"GCThreadCount"`
	Duration            float64          `json:"duration"`
	EndTime             float64          `json:"endTime"`
	Id                  int              `json:"id"`
	MemoryUsageAfterGc  []DetailMemoryKV `json:"MemoryUsageAfterGc"`
	MemoryUsageBeforeGc []DetailMemoryKV `json:"MemoryUsageBeforeGc"`
}

type DetailMemoryKV struct {
	Key   string       `json:"key"`
	Value DetailMemory `json:"value"`
}

type DetailMemory struct {
	Used      float64 `json:"used"`
	Max       float64 `json:"max"`
	Committed float64 `json:"committed"`
	Init      float64 `json:"init"`
}
