package models

type RepositoryList struct {
	Company      CompanyRepository
	User         UserRepository
	Agent        AgentRepository
	Contact      ContactRepository
	Conversation ConversationRepository
	Message      MessageRepository
	Template     TemplateRepository
	Billing      BillingRepository
}

func NewRepositoryList(
	company CompanyRepository,
	user UserRepository,
	agent AgentRepository,
	contact ContactRepository,
	conv ConversationRepository,
	msg MessageRepository,
	tmpl TemplateRepository,
	billing BillingRepository,
) *RepositoryList {
	return &RepositoryList{
		Company:      company,
		User:         user,
		Agent:        agent,
		Contact:      contact,
		Conversation: conv,
		Message:      msg,
		Template:     tmpl,
		Billing:      billing,
	}
}
