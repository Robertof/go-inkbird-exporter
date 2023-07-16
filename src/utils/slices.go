package utils

// props to: https://stackoverflow.com/a/28058324
func Reverse[S ~[]E, E any](s S) S {
  out := make(S, len(s))
  copy(out, s)

  for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
    out[i], out[j] = out[j], out[i]
  }

  return out
}
