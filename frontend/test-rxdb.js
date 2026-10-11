import { createRxDatabase } from 'rxdb';
import { getRxStorageWailsIPC } from './src/rxdb-ipc-storage_rxdb_ipc_002.js'; // Note we compile this later or use tsx

// Mock Wails backend
const mockDb = new Map();

global.window = {
  go: {
    main: {
      App: {
        GetAllDocs: async (collection) => {
          return mockDb.get(collection) || [];
        },
        BulkSaveDocs: async (collection, docs) => {
          let col = mockDb.get(collection) || [];
          for (const d of docs) {
             const existingIdx = col.findIndex(x => JSON.parse(x).id === d.id);
             if (existingIdx >= 0) {
                 col[existingIdx] = d.data;
             } else {
                 col.push(d.data);
             }
          }
          mockDb.set(collection, col);
        },
        RemoveCollection: async (collection) => {
          mockDb.delete(collection);
        }
      }
    }
  }
};

async function runTest() {
  const db = await createRxDatabase({
    name: 'testdb',
    storage: getRxStorageWailsIPC(),
    ignoreDuplicate: true
  });

  const schema = {
    version: 0,
    primaryKey: 'id',
    type: 'object',
    properties: {
      id: { type: 'string', maxLength: 100 },
      name: { type: 'string' }
    },
    required: ['id', 'name']
  };

  await db.addCollections({
    heroes: { schema }
  });

  console.log('Inserting document...');
  await db.heroes.insert({ id: 'h1', name: 'Batman' });
  
  console.log('Querying document...');
  const doc = await db.heroes.findOne('h1').exec();
  console.log('Found:', doc.toJSON());

  if (doc.name !== 'Batman') throw new Error('Test failed');

  console.log('Mock DB state:', mockDb);

  await db.destroy();
  console.log('Success! Test passed.');
}

runTest().catch(console.error);
