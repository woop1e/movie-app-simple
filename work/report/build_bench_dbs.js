const source = db.getSiblingDB('yozigo');
const demo = source.users.findOne({ email: 'student.demo@example.test' });
if (!demo) throw new Error('demo account is missing');
const demoSession = source.sessions.findOne({ userId: demo._id, expiresAt: { $gt: new Date() } });
if (!demoSession) throw new Error('demo session is missing or expired');
const demoEvents = source.events.find({ userId: demo._id }).toArray();
const sharedTitles = ['Interstellar', 'Inception', 'Arrival'];
const sharedMovieIds = sharedTitles.map(t => source.movies.findOne({ title: t })._id);
const sharedEvents = source.events.find({ source: 'MovieLens', movieId: { $in: sharedMovieIds } }).toArray();
const allUsers = source.users.find({ source: 'MovieLens' }).toArray();
const sizes = [
  { name: 'yozigo_bench_10k', ratings: 10000 },
  { name: 'yozigo_bench_50k', ratings: 50000 },
  { name: 'yozigo_bench_100k', ratings: 100836 },
];

function insertBatches(collection, docs) {
  for (let i = 0; i < docs.length; i += 1000) {
    collection.insertMany(docs.slice(i, i + 1000), { ordered: false });
  }
}

for (const size of sizes) {
  const selected = source.events.find({ source: 'MovieLens' }).sort({ _id: 1 }).limit(size.ratings).toArray();
  const eventById = new Map();
  for (const event of [...selected, ...sharedEvents, ...demoEvents]) eventById.set(event._id.toString(), event);
  const events = [...eventById.values()];
  const movieIDs = [...new Map(events.map(e => [e.movieId.toString(), e.movieId])).values()];
  for (const id of sharedMovieIds) if (!movieIDs.some(x => x.equals(id))) movieIDs.push(id);
  const movies = source.movies.find({ _id: { $in: movieIDs } }).toArray();
  const target = db.getSiblingDB(size.name);
  target.dropDatabase();
  insertBatches(target.movies, movies);
  insertBatches(target.users, [...allUsers, demo]);
  insertBatches(target.events, events);
  target.sessions.insertOne(demoSession);
  printjson({ database: size.name, movies: movies.length, users: allUsers.length + 1, events: events.length });
}
