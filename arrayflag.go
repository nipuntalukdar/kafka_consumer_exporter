package main

type arrayFlags []string

func (i *arrayFlags) String() (ret string) {
	for _, s := range []string(*i) {
		if ret != "" {
			ret += "," + s
		} else {
			ret = s
		}
	}
	return
}

func (a *arrayFlags) Set(value string) (err error) {
	*a = append(*a, value)
	return
}

func (a *arrayFlags) GetEls() []string {
	return []string(*a)
}

func (a *arrayFlags) GetElCount() int {
	return len([]string(*a))
}
