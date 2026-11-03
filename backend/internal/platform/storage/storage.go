package storage

import "os"

// Check verifies actual creation, write, sync and deletion, not only directory existence.
func Check(dir string) error {
	f,err:=os.CreateTemp(dir,".readiness-*");if err!=nil{return err}
	name:=f.Name()
	defer os.Remove(name)
	if _,err=f.Write([]byte("ready"));err!=nil{f.Close();return err}
	if err=f.Sync();err!=nil{f.Close();return err}
	if err=f.Close();err!=nil{return err}
	return os.Remove(name)
}
