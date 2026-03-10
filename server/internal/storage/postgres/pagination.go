package postgres

func calcTotalPages(total int64, pageSize int) int {
	if pageSize <= 0 {
		return 0
	}
	if total == 0 {
		return 0
	}

	pages := total / int64(pageSize)
	if total%int64(pageSize) != 0 {
		pages++
	}

	return int(pages)
}
