db.createUser({
  user: "aws-demo",
  pwd: "aws-demo",
  roles: ["dbOwner"]
});

db.bins.insertMany([
  { title: "firstBin", content: "firstBinContents" }
]);
