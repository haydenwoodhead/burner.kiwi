package server

func InboxMatcher(i Inbox) func(Inbox) bool {
	return func(e Inbox) bool {
		return i.Address == e.Address &&
			i.FailedToCreate == e.FailedToCreate &&
			i.CreatedBy == e.CreatedBy
	}
}
