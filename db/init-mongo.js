db.createUser({
  user: "pastebin",
  pwd:  "pastebin",
  roles: ["dbOwner"]
});

db["active-bins"].insertMany([
  { timestamp: 1743704875, title: "firstBin", content: "firstBinContents" }
]);
