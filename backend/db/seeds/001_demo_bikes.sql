-- 001_demo_bikes.sql
-- Idempotent seed data for 5 popular Indonesian motorcycles and their official service intervals

-- 1. Insert Master Bike Models
INSERT INTO bike_models (id, canonical_slug, brand, display_name, category, engine_cc, is_verified, source_provider)
VALUES 
    ('11111111-1111-4000-8000-000000000001', 'honda-vario-150', 'Honda', 'Honda Vario 150 eSP', 'matic', 150, true, 'manual_seed'),
    ('11111111-1111-4000-8000-000000000002', 'yamaha-nmax-155', 'Yamaha', 'Yamaha NMAX 155 Connected', 'matic', 155, true, 'manual_seed'),
    ('11111111-1111-4000-8000-000000000003', 'honda-beat-fi', 'Honda', 'Honda BeAT eSP 110', 'matic', 110, true, 'manual_seed'),
    ('11111111-1111-4000-8000-000000000004', 'yamaha-aerox-155', 'Yamaha', 'Yamaha Aerox 155 Connected', 'matic', 155, true, 'manual_seed'),
    ('11111111-1111-4000-8000-000000000005', 'kawasaki-ninja-250', 'Kawasaki', 'Kawasaki Ninja 250 FI (2-Cylinder)', 'sport', 250, true, 'manual_seed'),
    ('11111111-1111-4000-8000-000000000006', 'yamaha-mio-gear-125-s', 'Yamaha', 'Yamaha Gear 125 S-Version', 'matic', 125, true, 'manual_seed')
ON CONFLICT (canonical_slug) DO UPDATE SET
    brand = EXCLUDED.brand,
    display_name = EXCLUDED.display_name,
    category = EXCLUDED.category,
    engine_cc = EXCLUDED.engine_cc,
    is_verified = EXCLUDED.is_verified,
    source_provider = EXCLUDED.source_provider;

-- 2. Insert Master Spec Components
INSERT INTO bike_spec_components (bike_model_id, code, name, category, interval_km, interval_months, urgency_level, notes)
SELECT bm.id, comp.code, comp.name, comp.category, comp.interval_km, comp.interval_months, comp.urgency_level, comp.notes
FROM bike_models bm
JOIN (
    VALUES
    -- Honda Vario 150
    ('honda-vario-150', 'engine_oil', 'Oli Mesin SPX2 / MPX2', 'engine', 4000, 4, 'high', 'Kapasitas 0.8L SAE 10W-30'),
    ('honda-vario-150', 'transmission_oil', 'Oli Transmisi / Gardan', 'transmission', 8000, 8, 'medium', 'Kapasitas 120ml tiap 2x ganti oli mesin'),
    ('honda-vario-150', 'spark_plug', 'Busi NGK CPR9EA-9', 'ignition', 8000, 8, 'medium', 'Periksa celah busi 0.8-0.9mm'),
    ('honda-vario-150', 'air_filter', 'Filter Udara Viscous Paper', 'intake', 16000, 16, 'medium', 'Tipe elemen basah, dilarang disemprot angin kompresor'),
    ('honda-vario-150', 'cvt_belt', 'V-Belt & Roller CVT', 'transmission', 24000, 24, 'high', 'Paket penggantian v-belt dan weight roller'),
    ('honda-vario-150', 'brake_pad_front', 'Kampas Rem Cakram Depan', 'braking', 12000, 12, 'high', 'Periksa ketebalan kampas rem sebelum menyentuh batas aman'),
    ('honda-vario-150', 'coolant', 'Air Radiator / Coolant', 'cooling', 24000, 24, 'medium', 'Kuras dan ganti cairan pendingin radiator'),

    -- Yamaha NMAX 155
    ('yamaha-nmax-155', 'engine_oil', 'Oli Mesin Yamalube Super Matic', 'engine', 4000, 4, 'high', 'Kapasitas 0.9L / 1.0L SAE 10W-40 API SL JASO MB'),
    ('yamaha-nmax-155', 'transmission_oil', 'Oli Gear / Gardan Yamalube', 'transmission', 10000, 10, 'medium', 'Kapasitas 150ml oli gear transmisi'),
    ('yamaha-nmax-155', 'spark_plug', 'Busi NGK CPR8EA-9', 'ignition', 8000, 8, 'medium', 'Celah elektroda busi 0.8-0.9mm'),
    ('yamaha-nmax-155', 'air_filter', 'Filter Udara Kertas Basah', 'intake', 12000, 12, 'medium', 'Ganti baru, tidak dapat dicuci atau disemprot kompresor'),
    ('yamaha-nmax-155', 'cvt_belt', 'V-Belt & Slider CVT', 'transmission', 25000, 25, 'high', 'Pemeriksaan retak & keausan lebar sabuk v-belt'),
    ('yamaha-nmax-155', 'brake_fluid', 'Minyak Rem DOT 4 Depan & Belakang', 'braking', 20000, 24, 'high', 'Kuras dan ganti minyak rem sistem ABS'),
    ('yamaha-nmax-155', 'coolant', 'Cairan Radiator Yamacoolant', 'cooling', 20000, 24, 'medium', 'Kuras dan isi ulang coolant radiator'),

    -- Honda Beat FI
    ('honda-beat-fi', 'engine_oil', 'Oli Mesin AHM MPX2', 'engine', 4000, 4, 'high', 'Kapasitas 0.65L (eSP gen baru) / 0.8L SAE 10W-30'),
    ('honda-beat-fi', 'transmission_oil', 'Oli Transmisi AHM Gear Oil', 'transmission', 8000, 8, 'medium', 'Kapasitas 120ml tiap 8.000 km'),
    ('honda-beat-fi', 'spark_plug', 'Busi NGK MR9C-9N / Denso U27EPR-N9', 'ignition', 8000, 8, 'medium', 'Busi resistor khusus BeAT eSP'),
    ('honda-beat-fi', 'air_filter', 'Filter Udara Viscous Element', 'intake', 16000, 16, 'medium', 'Elemen kertas berperekat oli basah'),
    ('honda-beat-fi', 'cvt_belt', 'Drive Belt & Roller CVT', 'transmission', 24000, 24, 'high', 'Pemeriksaan berkala tiap 8.000 km, ganti tiap 24.000 km'),
    ('honda-beat-fi', 'brake_shoe_rear', 'Kampas Rem Tromol Belakang', 'braking', 12000, 12, 'high', 'Cek ketebalan kampas & setel jarak main bebas tuas rem'),

    -- Yamaha Aerox 155
    ('yamaha-aerox-155', 'engine_oil', 'Oli Mesin Yamalube Super Matic', 'engine', 4000, 4, 'high', 'Kapasitas 0.9L SAE 10W-40 API SL JASO MB'),
    ('yamaha-aerox-155', 'transmission_oil', 'Oli Gear Transmisi Yamalube', 'transmission', 10000, 10, 'medium', 'Kapasitas 150ml oli gear transmisi'),
    ('yamaha-aerox-155', 'spark_plug', 'Busi NGK CPR8EA-9', 'ignition', 8000, 8, 'medium', 'Pemeriksaan celah busi dan endapan karbon'),
    ('yamaha-aerox-155', 'air_filter', 'Filter Udara Viscous', 'intake', 12000, 12, 'medium', 'Elemen saringan udara basah tidak dapat dibersihkan'),
    ('yamaha-aerox-155', 'cvt_belt', 'V-Belt & Roller CVT', 'transmission', 25000, 25, 'high', 'Ganti sabuk transmisi dan roller pembeban'),
    ('yamaha-aerox-155', 'coolant', 'Cairan Pendingin Yamacoolant', 'cooling', 20000, 24, 'medium', 'Kuras cairan radiator sistem pendingin cair'),

    -- Kawasaki Ninja 250
    ('kawasaki-ninja-250', 'engine_oil', 'Oli Mesin KGO 4T Full Synthetic', 'engine', 3000, 4, 'high', 'Kapasitas 1.6L (tanpa filter) / 2.0L (dengan filter) SAE 10W-40'),
    ('kawasaki-ninja-250', 'oil_filter', 'Filter Oli Mesin Tabung', 'engine', 6000, 8, 'high', 'Ganti setiap 2x penggantian oli mesin'),
    ('kawasaki-ninja-250', 'spark_plug', 'Busi NGK CR9EIA-9 Iridium (2 pcs)', 'ignition', 12000, 12, 'medium', 'Busi tipe laser iridium untuk mesin 2-silinder paralel'),
    ('kawasaki-ninja-250', 'chain_lube_drive', 'Rantai Roda & Pelumasan Chain Lube', 'transmission', 1000, 1, 'high', 'Bersihkan dan lumasi rantai tiap 500-1000 km atau setelah hujan'),
    ('kawasaki-ninja-250', 'air_filter', 'Filter Udara Kertas Kering', 'intake', 12000, 12, 'medium', 'Bersihkan tiap 6.000 km, ganti baru tiap 12.000 km'),
    ('kawasaki-ninja-250', 'coolant', 'Kawasaki Genuine Coolant', 'cooling', 24000, 24, 'medium', 'Kuras cairan radiator mesin pendingin ganda'),
    ('kawasaki-ninja-250', 'brake_fluid', 'Minyak Rem DOT 4 Depan & Belakang', 'braking', 20000, 24, 'high', 'Kuras minyak rem hidrolik ganda'),

    -- Yamaha Mio Gear 125 S
    ('yamaha-mio-gear-125-s', 'engine_oil', 'Oli Mesin Yamalube Power Matic', 'engine', 4000, 4, 'high', 'Kapasitas 0.8L SAE 10W-40'),
    ('yamaha-mio-gear-125-s', 'transmission_oil', 'Oli Gear Transmisi Yamalube', 'transmission', 10000, 10, 'medium', 'Kapasitas 100ml oli gear'),
    ('yamaha-mio-gear-125-s', 'spark_plug', 'Busi NGK CR6HSA', 'ignition', 8000, 8, 'medium', 'Celah elektroda busi 0.6-0.7mm'),
    ('yamaha-mio-gear-125-s', 'air_filter', 'Filter Udara Viscous Kertas Basah', 'intake', 12000, 12, 'medium', 'Ganti baru, jangan disemprot angin'),
    ('yamaha-mio-gear-125-s', 'cvt_belt', 'V-Belt & Roller CVT', 'transmission', 24000, 24, 'high', 'Ganti satu set sabuk v-belt dan weight roller')
) AS comp(slug, code, name, category, interval_km, interval_months, urgency_level, notes)
ON bm.canonical_slug = comp.slug
ON CONFLICT (bike_model_id, code) DO UPDATE SET
    name = EXCLUDED.name,
    category = EXCLUDED.category,
    interval_km = EXCLUDED.interval_km,
    interval_months = EXCLUDED.interval_months,
    urgency_level = EXCLUDED.urgency_level,
    notes = EXCLUDED.notes;
