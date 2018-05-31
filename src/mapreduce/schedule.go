package mapreduce

import "fmt"

//
// schedule() starts and waits for all tasks in the given phase (mapPhase
// or reducePhase). the mapFiles argument holds the names of the files that
// are the inputs to the map phase, one per map task. nReduce is the
// number of reduce tasks. the registerChan argument yields a stream
// of registered workers; each item is the worker's RPC address,
// suitable for passing to call(). registerChan will yield all
// existing registered workers (if any) and new ones as they register.
//
func schedule(jobName string, mapFiles []string, nReduce int, phase jobPhase, registerChan chan string) {
	var ntasks int
	var nOther int // number of inputs (for reduce) or outputs (for map)
	switch phase { // nMap = len(mapFiles)
	case mapPhase:
		ntasks = len(mapFiles)
		nOther = nReduce
	case reducePhase:
		ntasks = nReduce
		nOther = len(mapFiles)
	}

	// All ntasks tasks have to be scheduled on workers. Once all tasks
	// have completed successfully, schedule() should return.

	fmt.Printf("Schedule start: %v %v tasks (%d I/Os)\n", ntasks, phase, nOther)
	// Your code here (Part III, Part IV).
	// Call signature: jobName, mapFiles, ntasks, nOther, phase, registerChan
	// ID conventions: workers by address (string); tasks by ID (int)
	idleWorkers := make(chan string)
	go func() { // forwards registerChan to idleWorkers
		for {
			idleWorkers <- <-registerChan
		}
	}()

	idleTasks := make(chan int, ntasks)
	doneTasks := make(chan int)
	doneCounter := 0
	for i := 0; i < ntasks; i++ {
		idleTasks <- i
	}

	for {
		select {
		case task := <-idleTasks:
			worker := <-idleWorkers
			taskArgs := DoTaskArgs{
				JobName:       jobName,
				Phase:         phase,
				TaskNumber:    task,
				NumOtherPhase: nOther,
			}
			if phase == "mapPhase" {
				taskArgs.File = mapFiles[task]
			}
			go do(task, worker, &taskArgs, idleWorkers, idleTasks, doneTasks)
		case <-doneTasks:
			doneCounter++
			if doneCounter == ntasks {
				fmt.Printf("Schedule finish: %v done\n", phase)
				return
			}
		}
	}
}

func do(task int, worker string, taskArgs *DoTaskArgs, idleWorkers chan string, idleTasks chan int, doneTasks chan int) {
	if call(worker, "Worker.DoTask", taskArgs, nil) {
		go func() { // idleWorkers only consumed if idleTasks consumed so this must not block
			idleWorkers <- worker
		}()
		doneTasks <- task
	} else {
		idleTasks <- task
	}
}
