import { getRxStorageMemory } from 'rxdb/plugins/storage-memory';
import type { RxStorage, RxStorageInstanceCreationParams } from 'rxdb';
// @ts-ignore
import { EventsOn } from '../wailsjs/runtime/runtime';

declare global {
  interface Window {
    go: {
      main: {
        App: {
          GetAllDocs(collection: string): Promise<string[]>;
          BulkSaveDocs(collection: string, docs: { id: string; data: string }[]): Promise<void>;
          RemoveCollection(collection: string): Promise<void>;
          GetNote(id: string): Promise<string>;
        };
      };
    };
  }
}

export function getRxStorageWailsIPC(): RxStorage<any, any> {
  const memoryStorage = getRxStorageMemory();
  
  return Object.assign({}, memoryStorage, {
    name: 'wails-ipc',
    async createStorageInstance<RxDocType>(params: RxStorageInstanceCreationParams<RxDocType, any>) {
      const instance = await memoryStorage.createStorageInstance(params);
      
      const collectionName = params.collectionName;
      const primaryPath = params.schema.primaryKey as string;

      try {
        if (window.go?.main?.App) {
          const allDocs = await window.go.main.App.GetAllDocs(collectionName);
          if (allDocs && allDocs.length > 0) {
            const documentWrites = allDocs.map(docStr => ({
              document: JSON.parse(docStr)
            }));
            await instance.bulkWrite(documentWrites, 'init');
          }
        }
      } catch (e) {
        console.error('Failed to load initial data', e);
      }
      
      try {
        EventsOn("notes:changed", async (ev: any) => {
          if (!window.go?.main?.App) return;
          if (collectionName !== "cards") return;
          
          const docId = ev.docId;
          const op = ev.op;
          
          if (op === "upsert") {
             const docStr = await window.go.main.App.GetNote(docId);
             if (docStr) {
               const doc = JSON.parse(docStr);
               
               // First fetch the old doc to provide the previous revision for overwrite
               let oldDoc = null;
               try {
                 const res = await instance.findDocumentsById([docId], false);
                 if (res && res[docId]) {
                   oldDoc = res[docId];
                 }
               } catch (e) {
               }
               
               await instance.bulkWrite([{
                 previous: oldDoc || undefined,
                 document: doc
               }], 'watcher');
             }
          } else if (op === "remove") {
               let oldDoc = null;
               try {
                 const res = await instance.findDocumentsById([docId], false);
                 if (res && res[docId]) {
                   oldDoc = res[docId];
                 }
               } catch (e) {
               }
               
               if (oldDoc) {
                 const delDoc = Object.assign({}, oldDoc, { _deleted: true });
                 await instance.bulkWrite([{
                   previous: oldDoc,
                   document: delDoc
                 }], 'watcher');
               }
          }
        });
      } catch(e) {
        console.error("Failed to register notes:changed listener", e);
      }

      const originalBulkWrite = instance.bulkWrite.bind(instance);
      instance.bulkWrite = async (documentWrites, context) => {
        const result = await originalBulkWrite(documentWrites, context);
        
        if (context !== 'watcher' && result.success && Object.keys(result.success).length > 0 && window.go?.main?.App) {
           const docsToSave = Object.values(result.success).map(d => ({
             id: d[primaryPath] as string,
             data: JSON.stringify(d)
           }));
           
           try {
             await window.go.main.App.BulkSaveDocs(collectionName, docsToSave);
           } catch (e) {}
        }
        return result;
      };

      const originalRemove = instance.remove ? instance.remove.bind(instance) : null;
      if (originalRemove) {
          instance.remove = async () => {
             if (window.go?.main?.App) {
                 await window.go.main.App.RemoveCollection(collectionName);
             }
             return originalRemove();
          }
      }

      return instance;
    }
  });
}
