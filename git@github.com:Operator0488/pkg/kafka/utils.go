package kafka

func convertSlice[S any, T any](
	input []S,
	convert func(S) T,
) []T {
	if len(input) == 0 {
		return []T{}
	}

	output := make([]T, len(input))
	for index, element := range input {
		output[index] = convert(element)
	}
	return output
}
