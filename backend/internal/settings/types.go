package settings

type ReferenceValue struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Label string `json:"label"`
}
type Checklist struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Items []string `json:"items"`
}
type Branding struct {
	Wordmark       string  `json:"wordmark"`
	LogoDocumentID *string `json:"logoDocumentId"`
}
type OperatingPolicy struct {
	Countries                []string `json:"countries"`
	NoticeVersion            *string  `json:"noticeVersion"`
	RetentionScheduleVersion *string  `json:"retentionScheduleVersion"`
	ConfirmedAt              *string  `json:"confirmedAt"`
}
type Settings struct {
	Version                  int64            `json:"version"`
	CompanyName              string           `json:"companyName"`
	Branding                 Branding         `json:"branding"`
	CurrencyCode             *string          `json:"currencyCode"`
	CurrencyMinorDigits      *int             `json:"currencyMinorDigits"`
	CurrencyLocked           bool             `json:"currencyLocked"`
	Timezone                 string           `json:"timezone"`
	TimezoneConfirmed        bool             `json:"timezoneConfirmed"`
	RecoveryInstructions     string           `json:"recoveryInstructions"`
	RecoveryContactLabel     string           `json:"recoveryContactLabel"`
	ReservationHours         int              `json:"reservationHours"`
	ReferenceValues          []ReferenceValue `json:"referenceValues"`
	InspectionChecklists     []Checklist      `json:"inspectionChecklists"`
	PropertyTypes            []string         `json:"propertyTypes"`
	OperatingPolicy          OperatingPolicy  `json:"operatingPolicy"`
	ConfirmOperationalPolicy bool             `json:"confirmOperationalPolicy,omitempty"`
}
