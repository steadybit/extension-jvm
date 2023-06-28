package extjvm

func InitTestJVM() {
  SpringApplications.Store(42, SpringApplication{
   Name: "customers",
   Pid:  int32(42),
   MvcMappings: &[]SpringMvcMapping{
     {
       Methods:     []string{"GET"},
       Patterns:    []string{"/customers"},
       HandlerClass: "com.steadybit.demo.CustomerController",
       HandlerName:  "customers",
     },
   },
  })
}
