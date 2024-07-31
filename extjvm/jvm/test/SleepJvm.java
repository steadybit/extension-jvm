class Scratch {
	public static void main(String[] args) throws Exception{
		var millis = Long.parseLong(args[0]);
		System.out.printf("Sleeping for %dms",millis);
		Thread.sleep(millis);
	}
}
