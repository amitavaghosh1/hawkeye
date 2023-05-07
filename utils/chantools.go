package utils

import "sync"

func JoinErrors(errChans ...chan error) chan error {
	out := make(chan error, 1)

	if len(errChans) == 1 {
		return errChans[0]
	}

	var wg sync.WaitGroup
	wg.Add(len(errChans))

	for _, errch := range errChans {
		go func(ch <-chan error) {
			for err := range ch {
				out <- err
			}
			wg.Done()
		}(errch)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
