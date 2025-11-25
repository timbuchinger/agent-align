const normalize = (message) => message.trim();

module.exports = {
  extends: ["@commitlint/config-conventional"],
  ignores: [
    (message) => normalize(message).startsWith("Merge branch "),
    (message) => normalize(message) === "Initial plan",
  ],
};
