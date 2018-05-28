package mapreduce

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

func doReduce(
	jobName string, // the name of the whole MapReduce job
	reduceTask int, // which reduce task this is
	outFile string, // write the output here
	nMap int, // the number of map tasks that were run ("M" in the paper)
	reduceF func(key string, values []string) string,
) {
	// 1. read nMap interm. files corresponding to the reduce task
	inputArray := make([]KeyValue, 0)
	for m := 0; m < nMap; m++ {
		// read file
		filename := reduceName(jobName, m, reduceTask)
		inputBytes, err := ioutil.ReadFile(filename)
		if err != nil {
			panic(err)
		}
		input := string(inputBytes)
		reader := strings.NewReader(input)

		dec := json.NewDecoder(reader)
		for {
			var kv KeyValue
			err := dec.Decode(&kv)
			if err != nil {
				break
			} else {
				inputArray = append(inputArray, kv)
			}
		}
	}
	// inputArray contains every result of the map tasks
	// corresponding to this reduce task, in KeyValue format

	// 2. sort interm. pairs by key
	sort.Slice(inputArray, func(i, j int) bool {
		return inputArray[i].Key > inputArray[j].Key
	})

	// 3. call reduceF on each key and write output to outFile as JSON

	// handles output
	file, err := os.Create(outFile)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	enc := json.NewEncoder(file)

	// processes inputArray
	var previousKey string
	var valueBuffer = make([]string, 0)
	for i, v := range inputArray {
		if v.Key != previousKey && i != 0 {
			result := KeyValue{previousKey, reduceF(previousKey, valueBuffer)}
			enc.Encode(&result)
			valueBuffer = make([]string, 0)
		}
		valueBuffer = append(valueBuffer, v.Value)
		previousKey = v.Key
	}
	result := KeyValue{previousKey, reduceF(previousKey, valueBuffer)}
	enc.Encode(&result)
	//
	// doReduce manages one reduce task: it should read the intermediate
	// files for the task, sort the intermediate key/value pairs by key,
	// call the user-defined reduce function (reduceF) for each key, and
	// write reduceF's output to disk.
	//
	// You'll need to read one intermediate file from each map task;
	// reduceName(jobName, m, reduceTask) yields the file
	// name from map task m.
	//
	// Your doMap() encoded the key/value pairs in the intermediate
	// files, so you will need to decode them. If you used JSON, you can
	// read and decode by creating a decoder and repeatedly calling
	// .Decode(&kv) on it until it returns an error.
	//
	// You may find the first example in the golang sort package
	// documentation useful.
	//
	// reduceF() is the application's reduce function. You should
	// call it once per distinct key, with a slice of all the values
	// for that key. reduceF() returns the reduced value for that key.
	//
	// You should write the reduce output as JSON encoded KeyValue
	// objects to the file named outFile. We require you to use JSON
	// because that is what the merger that combines the output
	// from all the reduce tasks expects. There is nothing special about
	// JSON -- it is just the marshalling format we chose to use. Your
	// output code will look something like this:
	//
	// enc := json.NewEncoder(file)
	// for key := ... {
	// 	enc.Encode(KeyValue{key, reduceF(...)})
	// }
	// file.Close()
	//
	// Your code here (Part I).
	//
}
