export interface PipeSnapshot {
  sourceId: string;
  revision: number;
  inventoryNumber: string;
  factoryNumber: string;
  manufacturer: string;
  diameterMm: number;
  wallThicknessMm: number;
  lengthMm: number;
  massKg: number | null;
  strengthClass: string;
  steelGrade: string;
  heatNumber: string;
  carbonEquivalentPct: number | null;
  pipeSpecification: string;
  coatingType: string;
  coatingSpecification: string;
  testPressureMpa: number | null;
  minOperatingTempC: number | null;
  incomingInspectionReference: string;
  allowedCategory54: string;
  allowedCategory74: string;
  notes: string;
  attributesJson: string;
  certificates: {
    id: string; reference: string; number: string; issuedOn: string;
    pageCount: number | null; issuer: string;
  }[];
}

export interface TransferredTask {
  id: string;
  revision: number;
  number: string;
  queuePosition: number;
  bendNumber: string;
  bendType: '1ГО' | '2ГО';
  targetAngleDeg: number;
  notes: string;
  createdAt: string;
  preparedAt: string;
  snapshotVersion: 1;
  pipeSnapshot: PipeSnapshot;
  machineSnapshot: {
    id: string; name: string; model: string; serialNumber: string; inventoryNumber: string;
  };
  configurationSnapshot: {
    sourceId: string; revision: number; name: string; version: string;
    technicalCardReference: string; diameterMm: number; bendType: string;
    targetAngle: string; minPipeLengthMm: number | null;
    parameters: { name: string; value: string; unit: string }[];
    notes: string;
  };
}

export interface TaskPackage {
  format: 'pipe-bending.tasks';
  schemaVersion: 1;
  tasks: TransferredTask[];
}

export interface TaskSummary {
  id: string; revision: number; number: string; queuePosition: number;
  bendNumber: string; targetAngleDeg: number; factoryNumber: string; machineModel: string;
}

export interface ImportResult {
  added: number;
  skipped: number;
  alreadyImported: boolean;
}
